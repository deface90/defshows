package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/testutil"
)

func TestNotificationRepository_PrefsAndDedup(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows")
	repo := repository.NewNotificationRepository(gdb)
	ctx := context.Background()

	user := &entity.User{Email: strptr("n@example.com"), Role: entity.RoleUser, Timezone: "UTC"}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// Defaults when unset.
	p, err := repo.GetPrefs(ctx, user.ID)
	if err != nil || p.EpisodeRelease || !p.SeasonStart || p.SeasonFinale || p.WeeklyDigest || !p.SocialFollows {
		t.Fatalf("default prefs: %v (%+v)", err, p)
	}
	// Upsert then read.
	p.EpisodeRelease = true
	p.LeadTimeHours = 48
	if err := repo.UpsertPrefs(ctx, p); err != nil {
		t.Fatalf("upsert prefs: %v", err)
	}
	got, _ := repo.GetPrefs(ctx, user.ID)
	if !got.EpisodeRelease || got.LeadTimeHours != 48 {
		t.Fatalf("prefs not persisted: %+v", got)
	}

	// Dedup by dedupe_key; a telegram channel enqueues a pending delivery.
	n := &entity.Notification{UserID: user.ID, Type: entity.NotifyEpisodeReleased, DedupeKey: "k1", Payload: "hi"}
	created, err := repo.EnqueueNotification(ctx, n, []string{"telegram"})
	if err != nil || !created {
		t.Fatalf("first enqueue: %v created=%v", err, created)
	}
	n2 := &entity.Notification{UserID: user.ID, Type: entity.NotifyEpisodeReleased, DedupeKey: "k1", Payload: "hi again"}
	created, err = repo.EnqueueNotification(ctx, n2, []string{"telegram"})
	if err != nil || created {
		t.Fatalf("dup enqueue: %v created=%v", err, created)
	}

	// Pending telegram requires a linked chat.
	if pend, _ := repo.Pending(ctx, "telegram", 10); len(pend) != 0 {
		t.Fatalf("want 0 pending without chat, got %d", len(pend))
	}
	if err := repository.NewUserRepository(gdb).SetTelegramChatID(ctx, user.ID, 555); err != nil {
		t.Fatalf("set chat: %v", err)
	}
	pend, _ := repo.Pending(ctx, "telegram", 10)
	if len(pend) != 1 || pend[0].Target != "555" {
		t.Fatalf("pending: %+v", pend)
	}

	// Mark sent removes it from pending.
	if err := repo.MarkSent(ctx, pend[0].ID); err != nil {
		t.Fatalf("mark sent: %v", err)
	}
	if pend, _ := repo.Pending(ctx, "telegram", 10); len(pend) != 0 {
		t.Fatalf("want 0 pending after sent, got %d", len(pend))
	}
}

func TestNotificationRepository_FeedOnlyNotDispatched(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows")
	repo := repository.NewNotificationRepository(gdb)
	ctx := context.Background()

	actor := &entity.User{Email: strptr("actor@example.com"), Role: entity.RoleUser, Timezone: "UTC"}
	target := &entity.User{Email: strptr("target@example.com"), Role: entity.RoleUser, Timezone: "UTC"}
	for _, u := range []*entity.User{actor, target} {
		if err := gdb.Create(u).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	// The target is Telegram-linked, so a telegram delivery WOULD dispatch — proving the
	// feed-only notification is withheld by having no delivery, not by a missing chat.
	if err := repository.NewUserRepository(gdb).SetTelegramChatID(ctx, target.ID, 777); err != nil {
		t.Fatalf("set chat: %v", err)
	}

	// Enqueue with no channels → feed-only (no deliveries).
	n := &entity.Notification{
		UserID: target.ID, Type: entity.NotifyFollowRequest, ActorID: &actor.ID,
		DedupeKey: "follow_request:1:2", Payload: "кто-то хочет подписаться",
	}
	if created, err := repo.EnqueueNotification(ctx, n, nil); err != nil || !created {
		t.Fatalf("enqueue feed-only: %v created=%v", err, created)
	}

	// No sender picks it up — there is no delivery row on any channel.
	for _, ch := range []string{"telegram", "apns", "fcm"} {
		if pend, _ := repo.Pending(ctx, ch, 10); len(pend) != 0 {
			t.Fatalf("%s sender must not pick up the feed-only notification, got %d", ch, len(pend))
		}
	}

	// But it IS in the in-app feed, with the actor preserved.
	feed, err := repo.ListForUser(ctx, target.ID, 10)
	if err != nil || len(feed) != 1 {
		t.Fatalf("feed: len=%d err=%v", len(feed), err)
	}
	if feed[0].ActorID == nil || *feed[0].ActorID != actor.ID || feed[0].Type != entity.NotifyFollowRequest {
		t.Fatalf("unexpected feed item: %+v", feed[0])
	}

	// DeleteByDedupeKeys clears it (re-fire support on unfollow/reject).
	if err := repo.DeleteByDedupeKeys(ctx, "follow_request:1:2"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if feed, _ := repo.ListForUser(ctx, target.ID, 10); len(feed) != 0 {
		t.Fatalf("want empty feed after delete, got %d", len(feed))
	}
}

func TestNotificationRepository_ReleaseCandidatesAndLinkTokens(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows")
	repo := repository.NewNotificationRepository(gdb)
	trackingRepo := repository.NewTrackingRepository(gdb)
	ctx := context.Background()

	chat := int64(42)
	user := &entity.User{Email: strptr("cand@example.com"), Role: entity.RoleUser, Timezone: "UTC", TelegramChatID: &chat}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	show := &entity.Show{TMDBID: 900, Title: "Aired Show", AiringStatus: entity.AiringNow}
	if err := gdb.Create(show).Error; err != nil {
		t.Fatalf("seed show: %v", err)
	}
	season := &entity.Season{ShowID: show.ID, SeasonNumber: 1, EpisodeCount: 1}
	gdb.Create(season)
	yesterday := time.Now().AddDate(0, 0, -1)
	ep := &entity.Episode{SeasonID: season.ID, ShowID: show.ID, SeasonNumber: 1, EpisodeNumber: 1, Name: "Pilot", AirDate: &yesterday}
	if err := gdb.Create(ep).Error; err != nil {
		t.Fatalf("seed ep: %v", err)
	}
	us := &entity.UserShow{UserID: user.ID, ShowID: show.ID, Status: entity.StatusWatching}
	if err := trackingRepo.AddUserShow(ctx, us); err != nil {
		t.Fatalf("add user show: %v", err)
	}

	since := time.Now().AddDate(0, 0, -7)
	if candidates, err := repo.ReleasedEpisodeCandidates(ctx, since); err != nil || len(candidates) != 0 {
		t.Fatalf("want no candidates with default prefs, got %+v, err=%v", candidates, err)
	}
	prefs := entity.DefaultNotificationPref(user.ID)
	prefs.EpisodeRelease = true
	if err := repo.UpsertPrefs(ctx, &prefs); err != nil {
		t.Fatalf("enable notifications: %v", err)
	}

	cands, err := repo.ReleasedEpisodeCandidates(ctx, since)
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}
	if len(cands) != 1 || cands[0].EpisodeID != ep.ID {
		t.Fatalf("want 1 candidate, got %+v", cands)
	}

	// After watching, no candidate.
	if err := trackingRepo.UpsertUserEpisode(ctx, &entity.UserEpisode{UserShowID: us.ID, EpisodeID: ep.ID, Watched: true}); err != nil {
		t.Fatalf("watch: %v", err)
	}
	if cands, _ := repo.ReleasedEpisodeCandidates(ctx, since); len(cands) != 0 {
		t.Fatalf("want 0 candidates after watch, got %d", len(cands))
	}

	// Link tokens.
	tok := &entity.TelegramLinkToken{Token: "tok1", UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)}
	if err := repo.CreateLinkToken(ctx, tok); err != nil {
		t.Fatalf("create token: %v", err)
	}
	uid, err := repo.ConsumeLinkToken(ctx, "tok1")
	if err != nil || uid != user.ID {
		t.Fatalf("consume: %v uid=%d", err, uid)
	}
	// One-time: second consume fails.
	if _, err := repo.ConsumeLinkToken(ctx, "tok1"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound on reuse, got %v", err)
	}
	// Expired token.
	exp := &entity.TelegramLinkToken{Token: "tok2", UserID: user.ID, ExpiresAt: time.Now().Add(-time.Minute)}
	_ = repo.CreateLinkToken(ctx, exp)
	if _, err := repo.ConsumeLinkToken(ctx, "tok2"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound for expired, got %v", err)
	}
}

func TestNotificationRepository_SeasonFinaleCandidates(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows")
	repo := repository.NewNotificationRepository(gdb)
	trackingRepo := repository.NewTrackingRepository(gdb)
	ctx := context.Background()

	chat := int64(42)
	user := &entity.User{Email: strptr("finale@example.com"), Role: entity.RoleUser, Timezone: "UTC", TelegramChatID: &chat}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	show := &entity.Show{TMDBID: 901, Title: "Finale Show", AiringStatus: entity.AiringNow}
	if err := gdb.Create(show).Error; err != nil {
		t.Fatalf("seed show: %v", err)
	}
	yesterday := time.Now().AddDate(0, 0, -1)
	tomorrow := time.Now().AddDate(0, 0, 1)
	// S1 fully aired (finale = e2); S2 still airing (finale not aired yet).
	s1 := &entity.Season{ShowID: show.ID, SeasonNumber: 1, EpisodeCount: 2}
	s2 := &entity.Season{ShowID: show.ID, SeasonNumber: 2, EpisodeCount: 2}
	gdb.Create(s1)
	gdb.Create(s2)
	eps := []*entity.Episode{
		{SeasonID: s1.ID, ShowID: show.ID, SeasonNumber: 1, EpisodeNumber: 1, Name: "S1E1", AirDate: &yesterday},
		{SeasonID: s1.ID, ShowID: show.ID, SeasonNumber: 1, EpisodeNumber: 2, Name: "S1E2", AirDate: &yesterday},
		{SeasonID: s2.ID, ShowID: show.ID, SeasonNumber: 2, EpisodeNumber: 1, Name: "S2E1", AirDate: &yesterday},
		{SeasonID: s2.ID, ShowID: show.ID, SeasonNumber: 2, EpisodeNumber: 2, Name: "S2E2", AirDate: &tomorrow},
	}
	for _, e := range eps {
		if err := gdb.Create(e).Error; err != nil {
			t.Fatalf("seed ep: %v", err)
		}
	}
	us := &entity.UserShow{UserID: user.ID, ShowID: show.ID, Status: entity.StatusWatching}
	if err := trackingRepo.AddUserShow(ctx, us); err != nil {
		t.Fatalf("add user show: %v", err)
	}

	since := time.Now().AddDate(0, 0, -7)
	if candidates, err := repo.SeasonFinaleCandidates(ctx, since); err != nil || len(candidates) != 0 {
		t.Fatalf("want no candidates with default prefs, got %+v, err=%v", candidates, err)
	}
	prefs := entity.DefaultNotificationPref(user.ID)
	prefs.SeasonFinale = true
	if err := repo.UpsertPrefs(ctx, &prefs); err != nil {
		t.Fatalf("enable notifications: %v", err)
	}

	cands, err := repo.SeasonFinaleCandidates(ctx, since)
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}
	// Only S1's finale (e2), not S2 (season not fully aired) nor S1E1 (not the finale).
	if len(cands) != 1 || cands[0].EpisodeID != eps[1].ID || cands[0].SeasonNumber != 1 {
		t.Fatalf("want 1 finale candidate for S1E2, got %+v", cands)
	}

	// Disabling season_finale suppresses the candidate.
	if err := repo.UpsertPrefs(ctx, &entity.NotificationPref{UserID: user.ID, EpisodeRelease: true, SeasonStart: true, SeasonFinale: false, Channel: "telegram", LeadTimeHours: 24}); err != nil {
		t.Fatalf("set prefs: %v", err)
	}
	if cands, _ := repo.SeasonFinaleCandidates(ctx, since); len(cands) != 0 {
		t.Fatalf("want 0 candidates when season_finale off, got %d", len(cands))
	}
}

func TestNotificationRepository_APNsChannel(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows")
	repo := repository.NewNotificationRepository(gdb)
	userRepo := repository.NewUserRepository(gdb)
	ctx := context.Background()

	user := &entity.User{Email: strptr("apns@example.com"), Role: entity.RoleUser, Timezone: "UTC"}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// Pending apns requires a linked device token.
	if pend, _ := repo.Pending(ctx, "apns", 10); len(pend) != 0 {
		t.Fatalf("want 0 pending without token, got %d", len(pend))
	}
	if err := userRepo.SetAPNsToken(ctx, user.ID, "device-token-1"); err != nil {
		t.Fatalf("set apns token: %v", err)
	}

	n := &entity.Notification{UserID: user.ID, Type: entity.NotifyEpisodeReleased, DedupeKey: "apns-k1", Payload: "hi"}
	if _, err := repo.EnqueueNotification(ctx, n, []string{"apns"}); err != nil {
		t.Fatalf("enqueue notification: %v", err)
	}
	pend, _ := repo.Pending(ctx, "apns", 10)
	if len(pend) != 1 || pend[0].Target != "device-token-1" || pend[0].Body != "hi" {
		t.Fatalf("pending: %+v", pend)
	}

	// A second user stealing the same device token unlinks it from the first.
	other := &entity.User{Email: strptr("apns2@example.com"), Role: entity.RoleUser, Timezone: "UTC"}
	if err := gdb.Create(other).Error; err != nil {
		t.Fatalf("seed other user: %v", err)
	}
	if err := userRepo.SetAPNsToken(ctx, other.ID, "device-token-1"); err != nil {
		t.Fatalf("steal apns token: %v", err)
	}
	tok, err := userRepo.APNsToken(ctx, user.ID)
	if err != nil {
		t.Fatalf("apns token: %v", err)
	}
	if tok != nil {
		t.Fatalf("want original owner's token cleared, got %v", *tok)
	}
}

func TestNotificationRepository_FCMChannel(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows")
	repo := repository.NewNotificationRepository(gdb)
	userRepo := repository.NewUserRepository(gdb)
	ctx := context.Background()

	user := &entity.User{Email: strptr("fcm@example.com"), Role: entity.RoleUser, Timezone: "UTC"}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// Pending fcm requires a linked device token.
	if pend, _ := repo.Pending(ctx, "fcm", 10); len(pend) != 0 {
		t.Fatalf("want 0 pending without token, got %d", len(pend))
	}
	if err := userRepo.SetFCMToken(ctx, user.ID, "fcm-device-token-1"); err != nil {
		t.Fatalf("set fcm token: %v", err)
	}

	n := &entity.Notification{UserID: user.ID, Type: entity.NotifyEpisodeReleased, DedupeKey: "fcm-k1", Payload: "hi"}
	if _, err := repo.EnqueueNotification(ctx, n, []string{"fcm"}); err != nil {
		t.Fatalf("enqueue notification: %v", err)
	}
	pend, _ := repo.Pending(ctx, "fcm", 10)
	if len(pend) != 1 || pend[0].Target != "fcm-device-token-1" || pend[0].Body != "hi" {
		t.Fatalf("pending: %+v", pend)
	}
}

func TestNotificationRepository_UpcomingCandidates(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows")
	repo := repository.NewNotificationRepository(gdb)
	trackingRepo := repository.NewTrackingRepository(gdb)
	ctx := context.Background()

	chat := int64(77)
	user := &entity.User{Email: strptr("upcoming@example.com"), Role: entity.RoleUser, Timezone: "UTC", TelegramChatID: &chat}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	show := &entity.Show{TMDBID: 901, Title: "Upcoming Show", AiringStatus: entity.AiringNow}
	if err := gdb.Create(show).Error; err != nil {
		t.Fatalf("seed show: %v", err)
	}
	season := &entity.Season{ShowID: show.ID, SeasonNumber: 2, EpisodeCount: 1}
	gdb.Create(season)
	// air_date is a DATE compared as midnight, so pin "now" 12h before the next
	// UTC midnight instead of using the wall clock (flaky before noon).
	y, m, d := time.Now().UTC().Date()
	soon := time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC)
	now := soon.Add(-12 * time.Hour)
	premiere := &entity.Episode{SeasonID: season.ID, ShowID: show.ID, SeasonNumber: 2, EpisodeNumber: 1, Name: "Premiere", AirDate: &soon}
	if err := gdb.Create(premiere).Error; err != nil {
		t.Fatalf("seed episode: %v", err)
	}
	us := &entity.UserShow{UserID: user.ID, ShowID: show.ID, Status: entity.StatusWatching}
	if err := trackingRepo.AddUserShow(ctx, us); err != nil {
		t.Fatalf("add user show: %v", err)
	}

	if candidates, err := repo.EpisodeUpcomingCandidates(ctx, now); err != nil || len(candidates) != 0 {
		t.Fatalf("want no candidates with default prefs, got %+v, err=%v", candidates, err)
	}
	prefs := entity.DefaultNotificationPref(user.ID)
	prefs.EpisodeRelease = true
	if err := repo.UpsertPrefs(ctx, &prefs); err != nil {
		t.Fatalf("enable notifications: %v", err)
	}

	epCands, err := repo.EpisodeUpcomingCandidates(ctx, now)
	if err != nil {
		t.Fatalf("episode upcoming: %v", err)
	}
	if len(epCands) != 1 || epCands[0].EpisodeID != premiere.ID {
		t.Fatalf("want 1 upcoming episode candidate, got %+v", epCands)
	}

	seasonCands, err := repo.SeasonUpcomingCandidates(ctx, now)
	if err != nil {
		t.Fatalf("season upcoming: %v", err)
	}
	if len(seasonCands) != 1 || seasonCands[0].EpisodeID != premiere.ID {
		t.Fatalf("want 1 upcoming season candidate, got %+v", seasonCands)
	}

	// Outside the default 24h lead time: no candidates.
	far := soon.Add(72 * time.Hour)
	premiere.AirDate = &far
	if err := gdb.Save(premiere).Error; err != nil {
		t.Fatalf("push air date: %v", err)
	}
	if cands, _ := repo.EpisodeUpcomingCandidates(ctx, now); len(cands) != 0 {
		t.Fatalf("want 0 candidates beyond lead time, got %d", len(cands))
	}
}
