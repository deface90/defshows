package tmdb

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	tmdbclient "github.com/deface90/defshows/backend/pkg/clients/tmdb"
	"github.com/deface90/defshows/backend/pkg/provider"
)

var _ provider.DiscoveryProvider = (*Provider)(nil)

func (p *Provider) Discover(ctx context.Context, opts provider.DiscoverOptions) (provider.DiscoveryPage, error) {
	editor := func(ctx context.Context, req *http.Request) error {
		if err := p.editor(ctx, req); err != nil {
			return err
		}
		q := req.URL.Query()
		for key, value := range map[string]string{
			"with_origin_country": opts.Country, "with_genres": opts.Genres,
			"first_air_date.gte": opts.DateFrom, "first_air_date.lte": opts.DateTo,
			"sort_by": opts.Sort, "page": strconv.Itoa(opts.Page),
			"vote_average.gte": strconv.FormatFloat(opts.RatingMin, 'f', -1, 64),
			"vote_average.lte": strconv.FormatFloat(opts.RatingMax, 'f', -1, 64),
			"vote_count.gte":   strconv.Itoa(opts.VotesMin), "include_adult": "false",
		} {
			if value != "" {
				q.Set(key, value)
			}
		}
		req.URL.RawQuery = q.Encode()
		return nil
	}
	resp, err := p.client.DiscoverTvWithResponse(ctx, editor)
	if err != nil {
		return provider.DiscoveryPage{}, fmt.Errorf("tmdb: discover: %w", safeRequestError(err))
	}
	if resp.JSON200 == nil {
		return provider.DiscoveryPage{}, fmt.Errorf("tmdb: discover: unexpected status %d", resp.StatusCode())
	}
	page := resp.JSON200
	return provider.DiscoveryPage{Results: summaries(page.Results), Page: val(page.Page), TotalPages: min(val(page.TotalPages), 500), TotalResults: val(page.TotalResults)}, nil
}

func summaries(results *[]tmdbclient.TvSearchResult) []provider.ShowSummary {
	out := []provider.ShowSummary{}
	if results == nil {
		return out
	}
	for _, r := range *results {
		out = append(out, provider.ShowSummary{
			TMDBID: val(r.Id), Title: val(r.Name), OriginalTitle: val(r.OriginalName),
			Overview: val(r.Overview), PosterURL: imageURL(r.PosterPath), FirstAirDate: parseDate(r.FirstAirDate),
			Popularity: float64(val(r.Popularity)), VoteAverage: float64(val(r.VoteAverage)), VoteCount: val(r.VoteCount),
		})
	}
	return out
}

// Cache the language-specific reference data on this provider instance for a day.
// Failed responses never replace the cache.
func (p *Provider) DiscoveryFilters(ctx context.Context) (provider.DiscoveryFilters, error) {
	p.filtersMu.Lock()
	defer p.filtersMu.Unlock()
	if time.Now().Before(p.filtersExpiry) {
		return p.filters, nil
	}
	genres, err := p.client.TvGenresWithResponse(ctx, p.editor)
	if err != nil {
		return provider.DiscoveryFilters{}, fmt.Errorf("tmdb: genres: %w", safeRequestError(err))
	}
	if genres.JSON200 == nil {
		return provider.DiscoveryFilters{}, fmt.Errorf("tmdb: genres: unexpected status %d", genres.StatusCode())
	}
	countries, err := p.client.CountriesWithResponse(ctx, p.editor)
	if err != nil {
		return provider.DiscoveryFilters{}, fmt.Errorf("tmdb: countries: %w", safeRequestError(err))
	}
	if countries.JSON200 == nil {
		return provider.DiscoveryFilters{}, fmt.Errorf("tmdb: countries: unexpected status %d", countries.StatusCode())
	}
	filters := provider.DiscoveryFilters{Genres: []provider.Genre{}, Countries: []provider.Country{}}
	for _, g := range genres.JSON200.Genres {
		filters.Genres = append(filters.Genres, provider.Genre{TMDBID: g.Id, Name: g.Name})
	}
	for _, c := range *countries.JSON200 {
		filters.Countries = append(filters.Countries, provider.Country{Code: c.Iso31661, Name: c.EnglishName})
	}
	p.filters, p.filtersExpiry = filters, time.Now().Add(24*time.Hour)
	return filters, nil
}
