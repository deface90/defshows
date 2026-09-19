package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/internal/testutil"
)

func TestTrackingUsecase_Flow(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows", "genres")
	ctx := context.Background()

	// A user.
	userRepo := repository.NewUserRepository(gdb)
	email := "tracker@example.com"
	user := &entity.User{Email: &email, Role: entity.RoleUser, Timezone: "UTC"}
	if err := userRepo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	catalogRepo := repository.NewCatalogRepository(gdb)
	catalogUC := usecase.NewCatalogUsecase(catalogRepo, sampleProvider())
	trackingRepo := repository.NewTrackingRepository(gdb)
	uc := usecase.NewTrackingUsecase(trackingRepo, catalogUC)

	// Add show (imports via provider on first use).
	us, err := uc.AddShow(ctx, user.ID, 1399)
	if err != nil {
		t.Fatalf("add show: %v", err)
	}
	if us.ShowID == 0 {
		t.Fatal("expected show id")
	}
	// Duplicate add.
	if _, err := uc.AddShow(ctx, user.ID, 1399); !errors.Is(err, usecase.ErrAlreadyTracked) {
		t.Fatalf("want ErrAlreadyTracked, got %v", err)
	}

	episodes, err := catalogRepo.GetEpisodesByShow(ctx, us.ShowID)
	if err != nil || len(episodes) != 2 {
		t.Fatalf("episodes: %v (n=%d)", err, len(episodes))
	}

	// Initial progress: 0/1, next = episode 1. Only episode 1 has aired (episode 2
	// has no air date), so exactly one aired-unwatched episode.
	prog, err := uc.GetProgress(ctx, user.ID, us.ShowID)
	if err != nil {
		t.Fatalf("progress: %v", err)
	}
	if prog.Watched != 0 || prog.Total != 1 || prog.Unwatched != 1 || prog.NextUnwatched == nil || prog.NextUnwatched.ID != episodes[0].ID {
		t.Fatalf("unexpected initial progress: %+v", prog)
	}

	// After watching the only aired episode (episode 1), no aired-unwatched left.
	if err := uc.SetEpisodeWatched(ctx, user.ID, us.ShowID, episodes[0].ID, true); err != nil {
		t.Fatalf("mark ep1: %v", err)
	}
	if prog, _ = uc.GetProgress(ctx, user.ID, us.ShowID); prog.Unwatched != 0 {
		t.Fatalf("expected 0 aired-unwatched after watching aired episode, got %+v", prog)
	}
	if err := uc.SetEpisodeWatched(ctx, user.ID, us.ShowID, episodes[0].ID, false); err != nil {
		t.Fatalf("unmark ep1: %v", err)
	}

	// Episodes can be watched out of order; progress must retain the exact IDs.
	if err := uc.SetEpisodeWatched(ctx, user.ID, us.ShowID, episodes[1].ID, true); err != nil {
		t.Fatal(err)
	}
	prog, err = uc.GetProgress(ctx, user.ID, us.ShowID)
	if err != nil || prog.Watched != 0 || len(prog.WatchedEpisodeIDs) != 1 || prog.WatchedEpisodeIDs[0] != episodes[1].ID || prog.NextUnwatched == nil || prog.NextUnwatched.ID != episodes[0].ID {
		t.Fatalf("out-of-order progress: %+v, %v", prog, err)
	}
	if err := uc.SetEpisodeWatched(ctx, user.ID, us.ShowID, episodes[1].ID, false); err != nil {
		t.Fatal(err)
	}

	// Mark episode 1 watched → 1/1, no released episode left.
	if err := uc.SetEpisodeWatched(ctx, user.ID, us.ShowID, episodes[0].ID, true); err != nil {
		t.Fatalf("mark watched: %v", err)
	}
	prog, _ = uc.GetProgress(ctx, user.ID, us.ShowID)
	if prog.Watched != 1 || prog.Total != 1 || prog.NextUnwatched != nil {
		t.Fatalf("progress after mark: %+v", prog)
	}

	// A legacy mark on an undated episode must not inflate progress.
	if err := uc.SetEpisodeWatched(ctx, user.ID, us.ShowID, episodes[1].ID, true); err != nil {
		t.Fatalf("mark ep2: %v", err)
	}
	prog, _ = uc.GetProgress(ctx, user.ID, us.ShowID)
	if prog.Watched != 1 || prog.Total != 1 || prog.NextUnwatched != nil {
		t.Fatalf("progress complete: %+v", prog)
	}

	// List shows with progress.
	list, err := uc.ListShows(ctx, user.ID, "")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v (n=%d)", err, len(list))
	}
	if list[0].Progress.Watched != 1 {
		t.Fatalf("list progress: %+v", list[0].Progress)
	}

	// Update status + dubbing.
	completed := string(entity.StatusCompleted)
	dub := "LostFilm"
	updated, err := uc.UpdateShow(ctx, user.ID, us.ShowID, &completed, nil, &dub)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Status != entity.StatusCompleted || updated.PreferredDubbing != "LostFilm" {
		t.Fatalf("update not applied: %+v", updated)
	}

	// Links.
	if _, err := uc.AddLink(ctx, user.ID, us.ShowID, "streaming", "LostFilm", "https://lostfilm.tv"); err != nil {
		t.Fatalf("add link: %v", err)
	}
	links, _ := uc.ListLinks(ctx, user.ID, us.ShowID)
	if len(links) != 1 {
		t.Fatalf("want 1 link, got %d", len(links))
	}

	// Not-tracked errors.
	if _, err := uc.GetProgress(ctx, user.ID, 999999); !errors.Is(err, usecase.ErrNotTracked) {
		t.Fatalf("want ErrNotTracked, got %v", err)
	}

	// Remove.
	if err := uc.RemoveShow(ctx, user.ID, us.ShowID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if l, _ := uc.ListShows(ctx, user.ID, ""); len(l) != 0 {
		t.Fatalf("want 0 shows after remove, got %d", len(l))
	}
}
