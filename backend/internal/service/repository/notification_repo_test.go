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
	if err != nil || !p.EpisodeRelease {
		t.Fatalf("default prefs: %v (%+v)", err, p)
	}
	// Upsert then read.
	p.EpisodeRelease = false
	p.LeadTimeHours = 48
	if err := repo.UpsertPrefs(ctx, p); err != nil {
		t.Fatalf("upsert prefs: %v", err)
	}
	got, _ := repo.GetPrefs(ctx, user.ID)
	if got.EpisodeRelease || got.LeadTimeHours != 48 {
		t.Fatalf("prefs not persisted: %+v", got)
	}

	// Dedup by dedupe_key.
	n := &entity.Notification{UserID: user.ID, Type: entity.NotifyEpisodeReleased, Channel: "telegram", Status: entity.NotifyPending, ScheduledFor: time.Now(), DedupeKey: "k1", Payload: "hi"}
	created, err := repo.CreateNotificationIfAbsent(ctx, n)
	if err != nil || !created {
		t.Fatalf("first create: %v created=%v", err, created)
	}
	n2 := &entity.Notification{UserID: user.ID, Type: entity.NotifyEpisodeReleased, Channel: "telegram", Status: entity.NotifyPending, ScheduledFor: time.Now(), DedupeKey: "k1", Payload: "hi again"}
	created, err = repo.CreateNotificationIfAbsent(ctx, n2)
	if err != nil || created {
		t.Fatalf("dup create: %v created=%v", err, created)
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

	n := &entity.Notification{UserID: user.ID, Type: entity.NotifyEpisodeReleased, Channel: "apns", Status: entity.NotifyPending, ScheduledFor: time.Now(), DedupeKey: "apns-k1", Payload: "hi"}
	if _, err := repo.CreateNotificationIfAbsent(ctx, n); err != nil {
		t.Fatalf("create notification: %v", err)
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
	soon := time.Now().Add(12 * time.Hour)
	premiere := &entity.Episode{SeasonID: season.ID, ShowID: show.ID, SeasonNumber: 2, EpisodeNumber: 1, Name: "Premiere", AirDate: &soon}
	if err := gdb.Create(premiere).Error; err != nil {
		t.Fatalf("seed episode: %v", err)
	}
	us := &entity.UserShow{UserID: user.ID, ShowID: show.ID, Status: entity.StatusWatching}
	if err := trackingRepo.AddUserShow(ctx, us); err != nil {
		t.Fatalf("add user show: %v", err)
	}

	now := time.Now()
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
	far := time.Now().Add(72 * time.Hour)
	premiere.AirDate = &far
	if err := gdb.Save(premiere).Error; err != nil {
		t.Fatalf("push air date: %v", err)
	}
	if cands, _ := repo.EpisodeUpcomingCandidates(ctx, now); len(cands) != 0 {
		t.Fatalf("want 0 candidates beyond lead time, got %d", len(cands))
	}
}
