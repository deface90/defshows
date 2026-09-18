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

func TestTrackingRepository(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows")
	ctx := context.Background()

	// Prerequisites: a user, a show, a season and two episodes.
	user := &entity.User{Email: strptr("track@example.com"), Role: entity.RoleUser, Timezone: "UTC"}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	show := &entity.Show{TMDBID: 555, Title: "Show", AiringStatus: entity.AiringEnded}
	if err := gdb.Create(show).Error; err != nil {
		t.Fatalf("seed show: %v", err)
	}
	season := &entity.Season{ShowID: show.ID, SeasonNumber: 1, Name: "S1", EpisodeCount: 2}
	if err := gdb.Create(season).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	ep1 := &entity.Episode{SeasonID: season.ID, ShowID: show.ID, SeasonNumber: 1, EpisodeNumber: 1, Name: "E1"}
	ep2 := &entity.Episode{SeasonID: season.ID, ShowID: show.ID, SeasonNumber: 1, EpisodeNumber: 2, Name: "E2"}
	if err := gdb.Create(ep1).Error; err != nil {
		t.Fatalf("seed ep1: %v", err)
	}
	if err := gdb.Create(ep2).Error; err != nil {
		t.Fatalf("seed ep2: %v", err)
	}

	repo := repository.NewTrackingRepository(gdb)

	us := &entity.UserShow{UserID: user.ID, ShowID: show.ID, Status: entity.StatusWatching}
	if err := repo.AddUserShow(ctx, us); err != nil {
		t.Fatalf("add user show: %v", err)
	}
	// Duplicate.
	if err := repo.AddUserShow(ctx, &entity.UserShow{UserID: user.ID, ShowID: show.ID, Status: entity.StatusWatching}); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}

	got, err := repo.GetUserShow(ctx, user.ID, show.ID)
	if err != nil || got.ID != us.ID {
		t.Fatalf("get user show: %v", err)
	}

	// Update.
	got.Status = entity.StatusCompleted
	got.Favorite = true
	got.PreferredDubbing = "LostFilm"
	if err := repo.UpdateUserShow(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	reread, _ := repo.GetUserShow(ctx, user.ID, show.ID)
	if reread.Status != entity.StatusCompleted || !reread.Favorite || reread.PreferredDubbing != "LostFilm" {
		t.Fatalf("update not applied: %+v", reread)
	}

	// List with status filter.
	list, _ := repo.ListUserShows(ctx, user.ID, "completed")
	if len(list) != 1 {
		t.Fatalf("want 1 completed, got %d", len(list))
	}
	if none, _ := repo.ListUserShows(ctx, user.ID, "watching"); len(none) != 0 {
		t.Fatalf("want 0 watching, got %d", len(none))
	}

	// Episode marks.
	if err := repo.UpsertUserEpisode(ctx, &entity.UserEpisode{UserShowID: us.ID, EpisodeID: ep1.ID, Watched: true}); err != nil {
		t.Fatalf("mark ep1: %v", err)
	}
	// Idempotent upsert.
	if err := repo.UpsertUserEpisode(ctx, &entity.UserEpisode{UserShowID: us.ID, EpisodeID: ep1.ID, Watched: true}); err != nil {
		t.Fatalf("re-mark ep1: %v", err)
	}
	ids, _ := repo.WatchedEpisodeIDs(ctx, us.ID)
	if len(ids) != 1 || ids[0] != ep1.ID {
		t.Fatalf("watched ids: %v", ids)
	}
	// Unmark.
	if err := repo.UpsertUserEpisode(ctx, &entity.UserEpisode{UserShowID: us.ID, EpisodeID: ep1.ID, Watched: false}); err != nil {
		t.Fatalf("unmark: %v", err)
	}
	if ids, _ := repo.WatchedEpisodeIDs(ctx, us.ID); len(ids) != 0 {
		t.Fatalf("want 0 watched after unmark, got %v", ids)
	}

	// Bulk mark preserves existing ratings and dates, and is scoped to the owner.
	watchedAt := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	rating := 8
	if err := repo.UpsertUserEpisode(ctx, &entity.UserEpisode{UserShowID: us.ID, EpisodeID: ep1.ID, Watched: true, WatchedAt: &watchedAt, Rating: &rating}); err != nil {
		t.Fatal(err)
	}
	if err := repo.WatchShow(ctx, user.ID+100000, show.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("foreign user: %v", err)
	}
	if ids, _ := repo.WatchedEpisodeIDs(ctx, us.ID); len(ids) != 1 {
		t.Fatal("foreign request changed marks")
	}
	for i := 0; i < 2; i++ {
		if err := repo.WatchShow(ctx, user.ID, show.ID); err != nil {
			t.Fatal(err)
		}
	}
	if ids, _ := repo.WatchedEpisodeIDs(ctx, us.ID); len(ids) != 2 {
		t.Fatalf("bulk watched IDs: %v", ids)
	}
	var saved entity.UserEpisode
	if err := gdb.Where("user_show_id = ? AND episode_id = ?", us.ID, ep1.ID).First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Rating == nil || *saved.Rating != rating || saved.WatchedAt == nil || !saved.WatchedAt.Equal(watchedAt) {
		t.Fatalf("bulk mark overwrote history: %+v", saved)
	}
	if err := repo.UpsertUserEpisode(ctx, &entity.UserEpisode{UserShowID: us.ID, EpisodeID: ep2.ID, Watched: false}); err != nil {
		t.Fatal(err)
	}
	if err := repo.WatchShow(ctx, user.ID, show.ID); err != nil {
		t.Fatal(err)
	}
	if ids, _ := repo.WatchedEpisodeIDs(ctx, us.ID); len(ids) != 2 {
		t.Fatalf("bulk mark failed to restore unwatched episode: %v", ids)
	}

	// Links.
	link := &entity.UserShowLink{UserShowID: us.ID, Kind: entity.LinkStreaming, Label: "LostFilm", URL: "https://x"}
	if err := repo.AddLink(ctx, link); err != nil {
		t.Fatalf("add link: %v", err)
	}
	links, _ := repo.ListLinks(ctx, us.ID)
	if len(links) != 1 {
		t.Fatalf("want 1 link, got %d", len(links))
	}
	if err := repo.UpdateLink(ctx, us.ID+100000, link.ID, "foreign"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("foreign link update: %v", err)
	}
	if err := repo.UpdateLink(ctx, us.ID, link.ID, "На диске"); err != nil {
		t.Fatal(err)
	}
	links, err = repo.ListLinks(ctx, us.ID)
	if err != nil || len(links) != 1 || links[0].ID != link.ID || links[0].URL != "На диске" || links[0].Label != "LostFilm" || links[0].Kind != entity.LinkStreaming {
		t.Fatalf("updated links: %+v, %v", links, err)
	}
	if err := repo.DeleteLink(ctx, us.ID, link.ID); err != nil {
		t.Fatalf("delete link: %v", err)
	}
	if links, _ := repo.ListLinks(ctx, us.ID); len(links) != 0 {
		t.Fatalf("want 0 links, got %d", len(links))
	}

	// Remove.
	if err := repo.RemoveUserShow(ctx, user.ID, show.ID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := repo.GetUserShow(ctx, user.ID, show.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound after remove, got %v", err)
	}
}
