package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/internal/testutil"
	"github.com/deface90/defshows/backend/pkg/provider"
)

// fakeProvider is a canned ShowProvider.
type fakeProvider struct {
	show      *provider.Show
	season    *provider.Season
	searchRes []provider.ShowSummary
	getShowN  int
}

func (f *fakeProvider) SearchShows(context.Context, string) ([]provider.ShowSummary, error) {
	return f.searchRes, nil
}

func (f *fakeProvider) GetShow(context.Context, int64) (*provider.Show, error) {
	f.getShowN++
	return f.show, nil
}

func (f *fakeProvider) GetSeason(context.Context, int64, int) (*provider.Season, error) {
	return f.season, nil
}

func sampleProvider() *fakeProvider {
	air := time.Date(2011, 4, 17, 0, 0, 0, 0, time.UTC)
	return &fakeProvider{
		show: &provider.Show{
			TMDBID:       1399,
			Title:        "Game of Thrones",
			Status:       "Ended",
			InProduction: false,
			Genres:       []provider.Genre{{TMDBID: 18, Name: "Drama"}},
			Seasons:      []provider.Season{{SeasonNumber: 1, Name: "Season 1", EpisodeCount: 2, AirDate: &air}},
			LastEpisode:  &provider.Episode{SeasonNumber: 8, EpisodeNumber: 6, Name: "The Iron Throne"},
		},
		season: &provider.Season{
			VoteAverage:  9,
			SeasonNumber: 1,
			Episodes: []provider.Episode{
				{SeasonNumber: 1, EpisodeNumber: 1, Name: "Winter Is Coming", AirDate: &air, VoteAverage: 8.5, VoteCount: 123},
				{SeasonNumber: 1, EpisodeNumber: 2, Name: "The Kingsroad"},
			},
		},
	}
}

func TestCatalogUsecase_ImportAndGet(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "shows", "genres")
	repo := repository.NewCatalogRepository(gdb)
	fp := sampleProvider()
	imdb, wiki := "https://www.imdb.com/title/tt0944947/", "https://ru.wikipedia.org/wiki/Game_of_Thrones"
	fp.show.IMDbURL, fp.show.WikipediaURL = &imdb, &wiki
	uc := usecase.NewCatalogUsecase(repo, fp)
	ctx := context.Background()

	show, err := uc.ImportShow(ctx, 1399)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if show.ID == 0 || show.AiringStatus != "ended" {
		t.Fatalf("unexpected show: %+v", show)
	}

	detail, err := uc.GetShow(ctx, show.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if detail.Show.IMDbURL == nil || *detail.Show.IMDbURL != imdb || detail.Show.WikipediaURL == nil || *detail.Show.WikipediaURL != wiki {
		t.Fatalf("reference links not persisted: %+v", detail.Show)
	}
	if len(detail.Show.Genres) != 1 {
		t.Fatalf("want 1 genre, got %d", len(detail.Show.Genres))
	}
	if len(detail.Show.Seasons) != 1 {
		t.Fatalf("want 1 season, got %d", len(detail.Show.Seasons))
	}
	if len(detail.Episodes) != 2 {
		t.Fatalf("want 2 episodes, got %d", len(detail.Episodes))
	}
	if detail.Show.NextEpisodeID != nil {
		t.Fatalf("expected nil next episode for ended show")
	}

	if detail.Show.Seasons[0].VoteAverage != 9 || detail.Episodes[0].VoteAverage != 8.5 || detail.Episodes[0].VoteCount != 123 {
		t.Fatalf("ratings not persisted: seasons=%+v episodes=%+v", detail.Show.Seasons, detail.Episodes)
	}
	// A refresh replaces ratings, including a rating cleared by the provider.
	fp.season.VoteAverage = 8
	fp.season.Episodes[0].VoteAverage = 0
	fp.season.Episodes[0].VoteCount = 0
	// Idempotent re-import: still one show, two episodes.
	if _, err := uc.ImportShow(ctx, 1399); err != nil {
		t.Fatalf("re-import: %v", err)
	}
	eps, _ := repo.GetEpisodesByShow(ctx, show.ID)
	if len(eps) != 2 {
		t.Fatalf("want 2 episodes after re-import, got %d", len(eps))
	}
	refreshed, err := uc.GetShow(ctx, show.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Show.Seasons[0].VoteAverage != 8 || refreshed.Episodes[0].VoteAverage != 0 || refreshed.Episodes[0].VoteCount != 0 {
		t.Fatalf("ratings not refreshed: %+v, %+v", refreshed.Show.Seasons, refreshed.Episodes)
	}
}

func TestCatalogUsecase_SkipsSpecials(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "shows", "genres")
	repo := repository.NewCatalogRepository(gdb)
	fp := sampleProvider()
	// Prepend a TMDB "Specials" season (season 0); it must be ignored on import.
	fp.show.Seasons = append([]provider.Season{{SeasonNumber: 0, Name: "Specials", EpisodeCount: 3}}, fp.show.Seasons...)
	uc := usecase.NewCatalogUsecase(repo, fp)
	ctx := context.Background()

	show, err := uc.ImportShow(ctx, 1399)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	detail, err := uc.GetShow(ctx, show.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(detail.Show.Seasons) != 1 {
		t.Fatalf("want 1 season (specials skipped), got %d", len(detail.Show.Seasons))
	}
	for _, s := range detail.Show.Seasons {
		if s.SeasonNumber == 0 {
			t.Fatalf("specials season was imported: %+v", s)
		}
	}
	for _, e := range detail.Episodes {
		if e.SeasonNumber == 0 {
			t.Fatalf("specials episode was imported: %+v", e)
		}
	}
	if len(detail.Episodes) != 2 {
		t.Fatalf("want 2 episodes, got %d", len(detail.Episodes))
	}
}

// fakeMirror records mirror calls and returns a deterministic public URL.
type fakeMirror struct {
	keys []string
	fail bool
}

func (m *fakeMirror) Mirror(_ context.Context, _, key string) (string, error) {
	m.keys = append(m.keys, key)
	if m.fail {
		return "", context.DeadlineExceeded
	}
	return "https://cdn.example/" + key, nil
}

func TestCatalogUsecase_ImportMirrorsImages(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "shows", "genres")
	repo := repository.NewCatalogRepository(gdb)
	fp := sampleProvider()
	fp.show.PosterURL = "https://image.tmdb.org/p/poster.jpg"
	fp.show.BackdropURL = "https://image.tmdb.org/p/backdrop.jpg"
	mirror := &fakeMirror{}
	uc := usecase.NewCatalogUsecase(repo, fp).WithImageMirror(mirror)
	ctx := context.Background()

	show, err := uc.ImportShow(ctx, 1399)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	// Deterministic keys, mirrored public URLs stored on the show.
	if want := []string{"shows/1399/poster.jpg", "shows/1399/backdrop.jpg"}; len(mirror.keys) != 2 || mirror.keys[0] != want[0] || mirror.keys[1] != want[1] {
		t.Fatalf("unexpected mirror keys: %v", mirror.keys)
	}
	if show.PosterKey == nil || *show.PosterKey != "https://cdn.example/shows/1399/poster.jpg" {
		t.Fatalf("poster not mirrored: %v", show.PosterKey)
	}
	if show.BackdropKey == nil || *show.BackdropKey != "https://cdn.example/shows/1399/backdrop.jpg" {
		t.Fatalf("backdrop not mirrored: %v", show.BackdropKey)
	}

	// On mirror failure, fall back to the original source URL.
	testutil.Truncate(t, gdb, "shows", "genres")
	fp2 := sampleProvider()
	fp2.show.PosterURL = "https://image.tmdb.org/p/poster.jpg"
	uc2 := usecase.NewCatalogUsecase(repo, fp2).WithImageMirror(&fakeMirror{fail: true})
	show2, err := uc2.ImportShow(ctx, 1399)
	if err != nil {
		t.Fatalf("import (mirror fail): %v", err)
	}
	if show2.PosterKey == nil || *show2.PosterKey != "https://image.tmdb.org/p/poster.jpg" {
		t.Fatalf("expected fallback to source URL, got %v", show2.PosterKey)
	}
}

func TestCatalogUsecase_EnsureShow(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "shows", "genres")
	repo := repository.NewCatalogRepository(gdb)
	fp := sampleProvider()
	uc := usecase.NewCatalogUsecase(repo, fp)
	ctx := context.Background()

	// First call imports (provider hit).
	if _, err := uc.EnsureShow(ctx, 1399); err != nil {
		t.Fatalf("ensure 1: %v", err)
	}
	// Second call uses the local mirror (no extra provider GetShow).
	before := fp.getShowN
	if _, err := uc.EnsureShow(ctx, 1399); err != nil {
		t.Fatalf("ensure 2: %v", err)
	}
	if fp.getShowN != before {
		t.Fatalf("expected no extra provider call, got %d (was %d)", fp.getShowN, before)
	}

	// Next-episode airing show → airing status + pointer set.
	air := time.Now().Add(48 * time.Hour)
	fp2 := sampleProvider()
	fp2.show.TMDBID = 999
	fp2.show.InProduction = true
	fp2.show.NextEpisode = &provider.Episode{SeasonNumber: 1, EpisodeNumber: 2, AirDate: &air}
	uc2 := usecase.NewCatalogUsecase(repo, fp2)
	s, err := uc2.ImportShow(ctx, 999)
	if err != nil {
		t.Fatalf("import airing: %v", err)
	}
	if s.AiringStatus != "airing" || s.NextEpisodeID == nil {
		t.Fatalf("expected airing with next episode, got status=%s next=%v", s.AiringStatus, s.NextEpisodeID)
	}
}
