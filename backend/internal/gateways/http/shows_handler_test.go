package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	httpapi "github.com/deface90/defshows/backend/internal/gateways/http"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/internal/testutil"
	"github.com/deface90/defshows/backend/pkg/provider"
	showsapi "github.com/deface90/defshows/backend/pkg/server/shows"
)

type fakeShowProvider struct{}

func (fakeShowProvider) SearchShows(context.Context, string) ([]provider.ShowSummary, error) {
	return []provider.ShowSummary{{TMDBID: 1399, Title: "Game of Thrones", Popularity: 100}}, nil
}

func (fakeShowProvider) GetShow(context.Context, int64) (*provider.Show, error) {
	air := time.Date(2011, 4, 17, 0, 0, 0, 0, time.UTC)
	return &provider.Show{
		TMDBID:  1399,
		Title:   "Game of Thrones",
		Status:  "Ended",
		Genres:  []provider.Genre{{TMDBID: 18, Name: "Drama"}},
		Seasons: []provider.Season{{SeasonNumber: 1, Name: "Season 1", EpisodeCount: 2, AirDate: &air}},
	}, nil
}

func (fakeShowProvider) GetSeason(context.Context, int64, int) (*provider.Season, error) {
	return &provider.Season{
		SeasonNumber: 1,
		Episodes: []provider.Episode{
			{SeasonNumber: 1, EpisodeNumber: 1, Name: "Winter Is Coming"},
			{SeasonNumber: 1, EpisodeNumber: 2, Name: "The Kingsroad"},
		},
	}, nil
}

func newShowsServer(t *testing.T) *echo.Echo {
	t.Helper()
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "shows", "genres")
	repo := repository.NewCatalogRepository(gdb)
	uc := usecase.NewCatalogUsecase(repo, fakeShowProvider{})
	e := echo.New()
	showsapi.RegisterHandlers(e, httpapi.NewShowsHandler(uc))
	return e
}

func TestShowsHandler_ImportGetListSearch(t *testing.T) {
	e := newShowsServer(t)

	// Search (external).
	rec := doJSON(t, e, http.MethodGet, "/shows/search?q=thrones", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("search: %d", rec.Code)
	}
	var sr showsapi.SearchResults
	_ = json.Unmarshal(rec.Body.Bytes(), &sr)
	if len(sr.Results) != 1 || sr.Results[0].TmdbId != 1399 {
		t.Fatalf("unexpected search: %+v", sr.Results)
	}

	// Import.
	rec = doJSON(t, e, http.MethodPost, "/shows/import", "", map[string]int64{"tmdb_id": 1399})
	if rec.Code != http.StatusOK {
		t.Fatalf("import: %d (%s)", rec.Code, rec.Body.String())
	}
	var show showsapi.Show
	if err := json.Unmarshal(rec.Body.Bytes(), &show); err != nil {
		t.Fatalf("decode import: %v", err)
	}
	if show.Id == 0 || show.Seasons == nil || len(*show.Seasons) != 1 {
		t.Fatalf("unexpected imported show: %+v", show)
	}
	if len((*show.Seasons)[0].Episodes) != 2 {
		t.Fatalf("want 2 nested episodes, got %d", len((*show.Seasons)[0].Episodes))
	}

	// Get by id.
	rec = doJSON(t, e, http.MethodGet, "/shows/"+strconv.FormatInt(show.Id, 10), "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: %d", rec.Code)
	}

	// Get missing → 404.
	rec = doJSON(t, e, http.MethodGet, "/shows/99999", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get missing: want 404, got %d", rec.Code)
	}

	// List local.
	rec = doJSON(t, e, http.MethodGet, "/shows?q=thrones", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d", rec.Code)
	}
	var list showsapi.ShowList
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Shows) != 1 {
		t.Fatalf("want 1 local show, got %d", len(list.Shows))
	}
}
