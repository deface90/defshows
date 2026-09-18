package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	httpapi "github.com/deface90/defshows/backend/internal/gateways/http"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/pkg/provider"
	showsapi "github.com/deface90/defshows/backend/pkg/server/shows"
	"github.com/labstack/echo/v4"
)

type discoverShowProvider struct {
	fakeShowProvider
	options provider.DiscoverOptions
	calls   int
}

func (p *discoverShowProvider) Discover(_ context.Context, opts provider.DiscoverOptions) (provider.DiscoveryPage, error) {
	p.options = opts
	p.calls++
	return provider.DiscoveryPage{Page: opts.Page, TotalPages: 3, TotalResults: 42, Results: []provider.ShowSummary{{TMDBID: 1399, Title: "Test", VoteAverage: 8.5, VoteCount: 500}}}, nil
}
func (p *discoverShowProvider) DiscoveryFilters(context.Context) (provider.DiscoveryFilters, error) {
	return provider.DiscoveryFilters{Genres: []provider.Genre{{TMDBID: 18, Name: "Драма"}}, Countries: []provider.Country{{Code: "US", Name: "United States"}}}, nil
}

func TestDiscoveryHTTP(t *testing.T) {
	p := &discoverShowProvider{}
	e := echo.New()
	showsapi.RegisterHandlers(e, httpapi.NewShowsHandler(usecase.NewCatalogUsecase(nil, p)))
	rec := doJSON(t, e, http.MethodGet, "/shows/discover?country=US&genres=18%7C35&rating_min=7.3&rating_max=9&votes_min=500&date_from=2000-01-01&date_to=2020-01-01&page=2&sort=vote_average.desc", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("discover: %d (%s)", rec.Code, rec.Body.String())
	}
	if p.options.Country != "US" || p.options.Genres != "18|35" || p.options.RatingMin != 7.3 || p.options.RatingMax != 9 || p.options.VotesMin != 500 || p.options.DateFrom != "2000-01-01" || p.options.DateTo != "2020-01-01" || p.options.Page != 2 || p.options.Sort != "vote_average.desc" {
		t.Fatalf("options: %+v", p.options)
	}
	var page showsapi.DiscoveryResults
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.TotalResults != 42 || page.Page != 2 || len(page.Results) != 1 || page.Results[0].TmdbId != 1399 {
		t.Fatalf("page: %+v", page)
	}
	rec = doJSON(t, e, http.MethodGet, "/shows/discover?rating_min=11", "", nil)
	if rec.Code != http.StatusBadRequest || p.calls != 1 {
		t.Fatalf("invalid filters: %d, calls %d", rec.Code, p.calls)
	}
	rec = doJSON(t, e, http.MethodGet, "/shows/discover/filters", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("references: %d", rec.Code)
	}
	var refs showsapi.DiscoveryFilters
	if err := json.Unmarshal(rec.Body.Bytes(), &refs); err != nil {
		t.Fatal(err)
	}
	if len(refs.Genres) != 1 || refs.Genres[0].Id != 18 || len(refs.Countries) != 1 || refs.Countries[0].Code != "US" {
		t.Fatalf("references: %+v", refs)
	}
}
