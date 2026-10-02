package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/internal/testutil"
)

func TestActivityBackfill_IdempotentSeed(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows", "genres", "activity_events", "user_shows", "user_episodes")
	ctx := context.Background()

	email := "backfill@example.com"
	user := &entity.User{Email: &email, Role: entity.RoleUser, Timezone: "UTC"}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	show := &entity.Show{TMDBID: 888, Title: "Backfillable", AiringStatus: entity.AiringEnded}
	if err := gdb.Create(show).Error; err != nil {
		t.Fatalf("seed show: %v", err)
	}
	season := &entity.Season{ShowID: show.ID, SeasonNumber: 1, Name: "S1", EpisodeCount: 2}
	if err := gdb.Create(season).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	aired1 := time.Now().AddDate(0, 0, -30)
	aired2 := time.Now().AddDate(0, 0, -20)
	future := time.Now().AddDate(0, 0, 30)
	ep1 := &entity.Episode{SeasonID: season.ID, ShowID: show.ID, SeasonNumber: 1, EpisodeNumber: 1, Name: "E1", AirDate: &aired1}
	ep2 := &entity.Episode{SeasonID: season.ID, ShowID: show.ID, SeasonNumber: 1, EpisodeNumber: 2, Name: "E2", AirDate: &aired2}
	ep3 := &entity.Episode{SeasonID: season.ID, ShowID: show.ID, SeasonNumber: 1, EpisodeNumber: 3, Name: "E3", AirDate: &future}
	for _, e := range []*entity.Episode{ep1, ep2, ep3} {
		if err := gdb.Create(e).Error; err != nil {
			t.Fatalf("seed episode: %v", err)
		}
	}
	addedAt := time.Now().AddDate(0, 0, -40)
	us := &entity.UserShow{UserID: user.ID, ShowID: show.ID, Status: entity.StatusWatching, AddedAt: addedAt}
	if err := gdb.Create(us).Error; err != nil {
		t.Fatalf("seed user_show: %v", err)
	}
	// Watch both aired episodes; ep3 hasn't aired and is left unwatched.
	for _, e := range []*entity.Episode{ep1, ep2} {
		wa := e.AirDate.AddDate(0, 0, 1)
		if err := gdb.Create(&entity.UserEpisode{UserShowID: us.ID, EpisodeID: e.ID, Watched: true, WatchedAt: &wa}).Error; err != nil {
			t.Fatalf("seed watch: %v", err)
		}
	}

	backfill := usecase.NewActivityBackfill(gdb)

	n, err := backfill.Run(ctx)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	// added_show(1) + watched_episode(2) + finished_season(1) + finished_show(1) = 5.
	if n != 5 {
		t.Fatalf("first run inserted %d, want 5", n)
	}

	count := func(typ entity.EventType) int64 {
		var c int64
		gdb.Model(&entity.ActivityEvent{}).Where("user_id = ? AND type = ?", user.ID, typ).Count(&c)
		return c
	}
	if count(entity.EventAddedShow) != 1 || count(entity.EventWatchedEpisode) != 2 ||
		count(entity.EventFinishedSeason) != 1 || count(entity.EventFinishedShow) != 1 {
		t.Fatalf("unexpected event counts: added=%d watched=%d season=%d show=%d",
			count(entity.EventAddedShow), count(entity.EventWatchedEpisode),
			count(entity.EventFinishedSeason), count(entity.EventFinishedShow))
	}

	// added_show created_at mirrors added_at; finished_show mirrors the latest watch.
	var addedEvt entity.ActivityEvent
	gdb.Where("user_id = ? AND type = ?", user.ID, entity.EventAddedShow).First(&addedEvt)
	if addedEvt.CreatedAt.Sub(addedAt).Abs() > time.Second {
		t.Fatalf("added_show created_at = %v, want ~%v", addedEvt.CreatedAt, addedAt)
	}

	// Second run is a pure no-op (idempotent).
	n2, err := backfill.Run(ctx)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("second run inserted %d, want 0 (idempotent)", n2)
	}
	var total int64
	gdb.Model(&entity.ActivityEvent{}).Where("user_id = ?", user.ID).Count(&total)
	if total != 5 {
		t.Fatalf("after re-run total = %d, want 5", total)
	}
}

func TestActivityBackfill_PartialWatchNoFinished(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows", "genres", "activity_events", "user_shows", "user_episodes")
	ctx := context.Background()

	email := "partial@example.com"
	user := &entity.User{Email: &email, Role: entity.RoleUser, Timezone: "UTC"}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	show := &entity.Show{TMDBID: 889, Title: "Partial", AiringStatus: entity.AiringEnded}
	if err := gdb.Create(show).Error; err != nil {
		t.Fatalf("seed show: %v", err)
	}
	season := &entity.Season{ShowID: show.ID, SeasonNumber: 1, Name: "S1", EpisodeCount: 2}
	if err := gdb.Create(season).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	aired1 := time.Now().AddDate(0, 0, -30)
	aired2 := time.Now().AddDate(0, 0, -20)
	ep1 := &entity.Episode{SeasonID: season.ID, ShowID: show.ID, SeasonNumber: 1, EpisodeNumber: 1, Name: "E1", AirDate: &aired1}
	ep2 := &entity.Episode{SeasonID: season.ID, ShowID: show.ID, SeasonNumber: 1, EpisodeNumber: 2, Name: "E2", AirDate: &aired2}
	for _, e := range []*entity.Episode{ep1, ep2} {
		if err := gdb.Create(e).Error; err != nil {
			t.Fatalf("seed episode: %v", err)
		}
	}
	us := &entity.UserShow{UserID: user.ID, ShowID: show.ID, Status: entity.StatusWatching, AddedAt: time.Now()}
	if err := gdb.Create(us).Error; err != nil {
		t.Fatalf("seed user_show: %v", err)
	}
	// Only one of two aired episodes watched → season/show incomplete.
	wa := aired1.AddDate(0, 0, 1)
	if err := gdb.Create(&entity.UserEpisode{UserShowID: us.ID, EpisodeID: ep1.ID, Watched: true, WatchedAt: &wa}).Error; err != nil {
		t.Fatalf("seed watch: %v", err)
	}

	if _, err := usecase.NewActivityBackfill(gdb).Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	var season0, show0 int64
	gdb.Model(&entity.ActivityEvent{}).Where("user_id = ? AND type = ?", user.ID, entity.EventFinishedSeason).Count(&season0)
	gdb.Model(&entity.ActivityEvent{}).Where("user_id = ? AND type = ?", user.ID, entity.EventFinishedShow).Count(&show0)
	if season0 != 0 || show0 != 0 {
		t.Fatalf("partial watch should not finish anything: season=%d show=%d", season0, show0)
	}
}
