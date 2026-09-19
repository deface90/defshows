package repository_test

import (
	"context"
	"testing"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/testutil"
)

func TestCatalogRepository_UpsertAndRelations(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "shows", "genres")
	repo := repository.NewCatalogRepository(gdb)
	ctx := context.Background()

	show := &entity.Show{
		TMDBID:       1399,
		Title:        "Game of Thrones",
		AiringStatus: entity.AiringEnded,
		Popularity:   100,
	}
	if err := repo.UpsertShow(ctx, show); err != nil {
		t.Fatalf("upsert show: %v", err)
	}
	firstID := show.ID
	if firstID == 0 {
		t.Fatal("expected generated show id")
	}

	// Idempotent upsert: same tmdb_id keeps the same row id.
	show2 := &entity.Show{TMDBID: 1399, Title: "Game of Thrones (updated)", AiringStatus: entity.AiringEnded}
	if err := repo.UpsertShow(ctx, show2); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	if show2.ID != firstID {
		t.Fatalf("want same id %d, got %d", firstID, show2.ID)
	}
	got, err := repo.GetShowByTMDBID(ctx, 1399)
	if err != nil {
		t.Fatalf("get by tmdb: %v", err)
	}
	if got.Title != "Game of Thrones (updated)" {
		t.Fatalf("update not applied: %q", got.Title)
	}

	// Seasons + episodes.
	seasons := []entity.Season{
		{ShowID: firstID, SeasonNumber: 1, Name: "Season 1", EpisodeCount: 2},
	}
	if err := repo.UpsertSeasons(ctx, seasons); err != nil {
		t.Fatalf("upsert seasons: %v", err)
	}
	seasonID := seasons[0].ID
	episodes := []entity.Episode{
		{SeasonID: seasonID, ShowID: firstID, SeasonNumber: 1, EpisodeNumber: 1, Name: "Winter Is Coming"},
		{SeasonID: seasonID, ShowID: firstID, SeasonNumber: 1, EpisodeNumber: 2, Name: "The Kingsroad"},
	}
	if err := repo.UpsertEpisodes(ctx, episodes); err != nil {
		t.Fatalf("upsert episodes: %v", err)
	}

	withRel, err := repo.GetShowWithRelations(ctx, firstID)
	if err != nil {
		t.Fatalf("get with relations: %v", err)
	}
	if len(withRel.Seasons) != 1 {
		t.Fatalf("want 1 season, got %d", len(withRel.Seasons))
	}
	eps, err := repo.GetEpisodesByShow(ctx, firstID)
	if err != nil || len(eps) != 2 {
		t.Fatalf("episodes: %v (n=%d)", err, len(eps))
	}

	// Re-upsert episodes is idempotent (still 2).
	if err := repo.UpsertEpisodes(ctx, episodes); err != nil {
		t.Fatalf("re-upsert episodes: %v", err)
	}
	eps, _ = repo.GetEpisodesByShow(ctx, firstID)
	if len(eps) != 2 {
		t.Fatalf("want 2 episodes after re-upsert, got %d", len(eps))
	}

	// next episode pointer.
	if err := repo.SetNextEpisode(ctx, firstID, &eps[1].ID, eps[1].AirDate, entity.AiringNow); err != nil {
		t.Fatalf("set next episode: %v", err)
	}
	got, _ = repo.GetShowByID(ctx, firstID)
	if got.NextEpisodeID == nil || *got.NextEpisodeID != eps[1].ID || got.AiringStatus != entity.AiringNow {
		t.Fatalf("next episode not set: %+v", got)
	}
}

func TestCatalogRepository_GenresAndSearch(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "shows", "genres")
	repo := repository.NewCatalogRepository(gdb)
	ctx := context.Background()

	show := &entity.Show{TMDBID: 42, Title: "The Expanse", Popularity: 50}
	if err := repo.UpsertShow(ctx, show); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	genres := []entity.Genre{{TMDBID: 10765, Name: "Sci-Fi & Fantasy"}, {TMDBID: 18, Name: "Drama"}}
	if err := repo.UpsertGenres(ctx, genres); err != nil {
		t.Fatalf("upsert genres: %v", err)
	}
	if genres[0].ID == 0 {
		t.Fatal("expected genre id populated")
	}
	if err := repo.ReplaceShowGenres(ctx, show.ID, []int64{genres[0].ID, genres[1].ID}); err != nil {
		t.Fatalf("link genres: %v", err)
	}
	withRel, _ := repo.GetShowWithRelations(ctx, show.ID)
	if len(withRel.Genres) != 2 {
		t.Fatalf("want 2 genres, got %d", len(withRel.Genres))
	}

	// Search.
	found, err := repo.ListShows(ctx, "expanse", 10, 0)
	if err != nil || len(found) != 1 {
		t.Fatalf("search: %v (n=%d)", err, len(found))
	}
	none, _ := repo.ListShows(ctx, "nonexistent-title", 10, 0)
	if len(none) != 0 {
		t.Fatalf("want 0 results, got %d", len(none))
	}
}

func TestCatalogRepository_Ratings(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "shows")
	repo := repository.NewCatalogRepository(gdb)
	ctx := context.Background()

	show := &entity.Show{TMDBID: 77, Title: "Rated"}
	if err := repo.UpsertShow(ctx, show); err != nil {
		t.Fatalf("upsert show: %v", err)
	}

	votes := int64(2000000)
	ratings := []entity.ShowRating{
		{ShowID: show.ID, Source: "imdb", Value: "9.0/10", Votes: &votes},
		{ShowID: show.ID, Source: "metacritic", Value: "85/100"},
	}
	if err := repo.UpsertRatings(ctx, ratings); err != nil {
		t.Fatalf("upsert ratings: %v", err)
	}

	// Upsert again with a changed value → still 2 rows, value updated.
	ratings[0].Value = "9.1/10"
	if err := repo.UpsertRatings(ctx, ratings); err != nil {
		t.Fatalf("re-upsert ratings: %v", err)
	}
	got, err := repo.GetRatingsByShow(ctx, show.ID)
	if err != nil || len(got) != 2 {
		t.Fatalf("get ratings: %v (n=%d)", err, len(got))
	}
	for _, r := range got {
		if r.Source == "imdb" && r.Value != "9.1/10" {
			t.Fatalf("imdb value not updated: %q", r.Value)
		}
	}
}

func TestCatalogRepository_ReferenceLinksRefresh(t *testing.T) {
	db := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, db, "shows", "genres")
	repo := repository.NewCatalogRepository(db)
	imdb, wiki := "https://www.imdb.com/title/tt0944947/", "https://en.wikipedia.org/wiki/Game_of_Thrones"
	ctx := t.Context()
	show := &entity.Show{TMDBID: 1399, Title: "Game of Thrones", IMDbURL: &imdb, WikipediaURL: &wiki}
	if err := repo.UpsertShow(ctx, show); err != nil {
		t.Fatal(err)
	}
	// Failed enrichment must not erase known links.
	if err := repo.UpsertShow(ctx, &entity.Show{TMDBID: 1399, Title: "Updated"}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetShowByID(ctx, show.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.IMDbURL == nil || *got.IMDbURL != imdb || got.WikipediaURL == nil || *got.WikipediaURL != wiki {
		t.Fatalf("lost links: %+v", got)
	}
	// A successful lookup can replace a URL or explicitly remove a stale link.
	updated, empty := "https://ru.wikipedia.org/wiki/Game_of_Thrones", ""
	if err := repo.UpsertShow(ctx, &entity.Show{TMDBID: 1399, Title: "Updated", IMDbURL: &empty, WikipediaURL: &updated}); err != nil {
		t.Fatal(err)
	}
	got, err = repo.GetShowByID(ctx, show.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.IMDbURL == nil || *got.IMDbURL != "" || got.WikipediaURL == nil || *got.WikipediaURL != updated {
		t.Fatalf("links not refreshed: %+v", got)
	}
}
