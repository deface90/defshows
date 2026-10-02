package usecase_test

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/internal/testutil"
	"github.com/deface90/defshows/backend/pkg/provider"
)

// multiSeasonProvider is a canned ShowProvider with two aired seasons so emission
// of finished_season can be distinguished from finished_show: season 1 has two
// episodes, season 2 has one, and all three have aired.
type multiSeasonProvider struct{ seasons map[int]*provider.Season }

func newMultiSeasonProvider() *multiSeasonProvider {
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	return &multiSeasonProvider{seasons: map[int]*provider.Season{
		1: {SeasonNumber: 1, Name: "Season 1", EpisodeCount: 2, AirDate: &past, Episodes: []provider.Episode{
			{SeasonNumber: 1, EpisodeNumber: 1, Name: "S1E1", AirDate: &past},
			{SeasonNumber: 1, EpisodeNumber: 2, Name: "S1E2", AirDate: &past},
		}},
		2: {SeasonNumber: 2, Name: "Season 2", EpisodeCount: 1, AirDate: &past, Episodes: []provider.Episode{
			{SeasonNumber: 2, EpisodeNumber: 1, Name: "S2E1", AirDate: &past},
		}},
	}}
}

func (p *multiSeasonProvider) SearchShows(context.Context, string) ([]provider.ShowSummary, error) {
	return nil, nil
}

func (p *multiSeasonProvider) GetShow(context.Context, int64) (*provider.Show, error) {
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	return &provider.Show{
		TMDBID: 2000, Title: "Two Seasons", Status: "Ended", FirstAirDate: &past,
		Seasons: []provider.Season{*p.seasons[1], *p.seasons[2]},
	}, nil
}

func (p *multiSeasonProvider) GetSeason(_ context.Context, _ int64, seasonNumber int) (*provider.Season, error) {
	return p.seasons[seasonNumber], nil
}

// eventDrain reads newly-inserted activity events for a user since the last call.
type eventDrain struct {
	gdb    *gorm.DB
	userID int64
	seen   int64
}

func (d *eventDrain) next(t *testing.T) []entity.ActivityEvent {
	t.Helper()
	var events []entity.ActivityEvent
	if err := d.gdb.Where("user_id = ? AND id > ?", d.userID, d.seen).
		Order("id").Find(&events).Error; err != nil {
		t.Fatalf("read events: %v", err)
	}
	if n := len(events); n > 0 {
		d.seen = events[n-1].ID
	}
	return events
}

func types(events []entity.ActivityEvent) []entity.EventType {
	out := make([]entity.EventType, len(events))
	for i, e := range events {
		out[i] = e.Type
	}
	return out
}

func eqTypes(a []entity.EventType, b ...entity.EventType) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTrackingUsecase_EmitsActivityEvents(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows", "genres", "activity_events", "user_shows", "user_episodes")
	ctx := context.Background()

	userRepo := repository.NewUserRepository(gdb)
	email := "emitter@example.com"
	user := &entity.User{Email: &email, Role: entity.RoleUser, Timezone: "UTC"}
	if err := userRepo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	catalogUC := usecase.NewCatalogUsecase(repository.NewCatalogRepository(gdb), newMultiSeasonProvider())
	uc := usecase.NewTrackingUsecase(repository.NewTrackingRepository(gdb), catalogUC)
	drain := &eventDrain{gdb: gdb, userID: user.ID}

	// Add show → one added_show event carrying the show id.
	us, err := uc.AddShow(ctx, user.ID, 2000)
	if err != nil {
		t.Fatalf("add show: %v", err)
	}
	if evs := drain.next(t); !eqTypes(types(evs), entity.EventAddedShow) || evs[0].ShowID != us.ShowID {
		t.Fatalf("add show events = %+v", evs)
	}

	eps, err := repository.NewCatalogRepository(gdb).GetEpisodesByShow(ctx, us.ShowID)
	if err != nil || len(eps) != 3 {
		t.Fatalf("episodes: %v (n=%d)", err, len(eps))
	}
	epID := func(season, number int) int64 {
		for _, e := range eps {
			if e.SeasonNumber == season && e.EpisodeNumber == number {
				return e.ID
			}
		}
		t.Fatalf("episode S%02dE%02d not found", season, number)
		return 0
	}

	// First episode of season 1: watched_episode only (season still 1/2).
	if err := uc.SetEpisodeWatched(ctx, user.ID, us.ShowID, epID(1, 1), true); err != nil {
		t.Fatal(err)
	}
	if evs := drain.next(t); !eqTypes(types(evs), entity.EventWatchedEpisode) {
		t.Fatalf("S1E1 events = %v", types(evs))
	}

	// Completing season 1 → watched_episode + finished_season (show still incomplete).
	if err := uc.SetEpisodeWatched(ctx, user.ID, us.ShowID, epID(1, 2), true); err != nil {
		t.Fatal(err)
	}
	evs := drain.next(t)
	if !eqTypes(types(evs), entity.EventWatchedEpisode, entity.EventFinishedSeason) {
		t.Fatalf("S1E2 events = %v", types(evs))
	}
	if evs[1].SeasonNumber == nil || *evs[1].SeasonNumber != 1 {
		t.Fatalf("finished_season should carry season 1, got %+v", evs[1].SeasonNumber)
	}

	// Last remaining episode → watched_episode + finished_season(2) + finished_show.
	if err := uc.SetEpisodeWatched(ctx, user.ID, us.ShowID, epID(2, 1), true); err != nil {
		t.Fatal(err)
	}
	if evs := drain.next(t); !eqTypes(types(evs),
		entity.EventWatchedEpisode, entity.EventFinishedSeason, entity.EventFinishedShow) {
		t.Fatalf("S2E1 events = %v", types(evs))
	}

	// Un-marking emits nothing.
	if err := uc.SetEpisodeWatched(ctx, user.ID, us.ShowID, epID(2, 1), false); err != nil {
		t.Fatal(err)
	}
	if evs := drain.next(t); len(evs) != 0 {
		t.Fatalf("unwatch should emit nothing, got %v", types(evs))
	}

	// Re-marking an already-watched episode is a no-op (emit nothing).
	if err := uc.SetEpisodeWatched(ctx, user.ID, us.ShowID, epID(1, 1), true); err != nil {
		t.Fatal(err)
	}
	if evs := drain.next(t); len(evs) != 0 {
		t.Fatalf("re-mark should emit nothing, got %v", types(evs))
	}

	// Rating: set emits rated_show (with the value); same value is a no-op; a change
	// emits again; a clear emits nothing.
	eight := 8
	if _, err := uc.UpdateShow(ctx, user.ID, us.ShowID, nil, nil, nil, &eight, false); err != nil {
		t.Fatal(err)
	}
	evs = drain.next(t)
	if !eqTypes(types(evs), entity.EventRatedShow) || evs[0].Rating == nil || *evs[0].Rating != 8 {
		t.Fatalf("set rating events = %+v", evs)
	}
	if _, err := uc.UpdateShow(ctx, user.ID, us.ShowID, nil, nil, nil, &eight, false); err != nil {
		t.Fatal(err)
	}
	if evs := drain.next(t); len(evs) != 0 {
		t.Fatalf("re-set same rating should emit nothing, got %v", types(evs))
	}
	nine := 9
	if _, err := uc.UpdateShow(ctx, user.ID, us.ShowID, nil, nil, nil, &nine, false); err != nil {
		t.Fatal(err)
	}
	if evs := drain.next(t); !eqTypes(types(evs), entity.EventRatedShow) {
		t.Fatalf("change rating events = %v", types(evs))
	}
	if _, err := uc.UpdateShow(ctx, user.ID, us.ShowID, nil, nil, nil, nil, true); err != nil {
		t.Fatal(err)
	}
	if evs := drain.next(t); len(evs) != 0 {
		t.Fatalf("clear rating should emit nothing, got %v", types(evs))
	}
}

func TestTrackingUsecase_WatchShowEmitsFinishedOnly(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows", "genres", "activity_events", "user_shows", "user_episodes")
	ctx := context.Background()

	userRepo := repository.NewUserRepository(gdb)
	email := "binger@example.com"
	user := &entity.User{Email: &email, Role: entity.RoleUser, Timezone: "UTC"}
	if err := userRepo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	catalogUC := usecase.NewCatalogUsecase(repository.NewCatalogRepository(gdb), newMultiSeasonProvider())
	uc := usecase.NewTrackingUsecase(repository.NewTrackingRepository(gdb), catalogUC)
	drain := &eventDrain{gdb: gdb, userID: user.ID}

	us, err := uc.AddShow(ctx, user.ID, 2000)
	if err != nil {
		t.Fatalf("add show: %v", err)
	}
	_ = drain.next(t) // discard the added_show event

	if err := uc.WatchShow(ctx, user.ID, us.ShowID); err != nil {
		t.Fatalf("watch show: %v", err)
	}
	evs := drain.next(t)
	// Two aired seasons → finished_season(1), finished_season(2), finished_show; and
	// crucially NO per-episode watched_episode rows from the bulk watch.
	if !eqTypes(types(evs),
		entity.EventFinishedSeason, entity.EventFinishedSeason, entity.EventFinishedShow) {
		t.Fatalf("watch show events = %v", types(evs))
	}
	if evs[0].SeasonNumber == nil || *evs[0].SeasonNumber != 1 || evs[1].SeasonNumber == nil || *evs[1].SeasonNumber != 2 {
		t.Fatalf("finished_season numbers = %v, %v", evs[0].SeasonNumber, evs[1].SeasonNumber)
	}
}
