// Package tmdb implements provider.ShowProvider on top of the generated TMDB
// client.
package tmdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	tmdbclient "github.com/deface90/defshows/backend/pkg/clients/tmdb"
	"github.com/deface90/defshows/backend/pkg/provider"
)

const (
	defaultBaseURL = "https://api.themoviedb.org/3"
	imageBaseURL   = "https://image.tmdb.org/t/p/w500"
)

// Provider is the TMDB-backed ShowProvider.
type Provider struct {
	filtersMu     sync.Mutex
	filters       provider.DiscoveryFilters
	filtersExpiry time.Time
	client        *tmdbclient.ClientWithResponses
	apiKey        string
	language      string
}

var _ provider.ShowProvider = (*Provider)(nil)

// New creates a TMDB provider. baseURL and httpClient may be empty/nil for
// production defaults; they are overridable for testing.
func New(baseURL, apiKey, language string, httpClient *http.Client) (*Provider, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if language == "" {
		language = "en-US"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	opts := []tmdbclient.ClientOption{}
	if httpClient != nil {
		opts = append(opts, tmdbclient.WithHTTPClient(httpClient))
	}
	c, err := tmdbclient.NewClientWithResponses(baseURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("tmdb: new client: %w", err)
	}
	return &Provider{client: c, apiKey: apiKey, language: language}, nil
}

// editor injects api_key and language on every request.
func (p *Provider) editor(_ context.Context, req *http.Request) error {
	q := req.URL.Query()
	q.Set("api_key", p.apiKey)
	q.Set("language", p.language)
	req.URL.RawQuery = q.Encode()
	return nil
}

// SearchShows searches TMDB for series.
func (p *Provider) SearchShows(ctx context.Context, query string) ([]provider.ShowSummary, error) {
	resp, err := p.client.SearchTvWithResponse(ctx, &tmdbclient.SearchTvParams{Query: query}, p.editor)
	if err != nil {
		return nil, fmt.Errorf("tmdb: search: %w", safeRequestError(err))
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("tmdb: search: unexpected status %d", resp.StatusCode())
	}
	return summaries(resp.JSON200.Results), nil
}

// GetShow fetches full series detail.
func (p *Provider) GetShow(ctx context.Context, tmdbID int64) (*provider.Show, error) {
	resp, err := p.client.GetTvShowWithResponse(ctx, tmdbID, p.editor)
	if err != nil {
		return nil, fmt.Errorf("tmdb: get show: %w", safeRequestError(err))
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("tmdb: get show: unexpected status %d", resp.StatusCode())
	}
	s := resp.JSON200
	show := &provider.Show{
		TMDBID:           val(s.Id),
		Title:            val(s.Name),
		OriginalTitle:    val(s.OriginalName),
		Overview:         val(s.Overview),
		PosterURL:        imageURL(s.PosterPath),
		BackdropURL:      imageURL(s.BackdropPath),
		Status:           val(s.Status),
		InProduction:     val(s.InProduction),
		FirstAirDate:     parseDate(s.FirstAirDate),
		LastAirDate:      parseDate(s.LastAirDate),
		OriginalLanguage: val(s.OriginalLanguage),
		Popularity:       float64(val(s.Popularity)),
		VoteAverage:      float64(val(s.VoteAverage)),
		VoteCount:        val(s.VoteCount),
		NextEpisode:      mapEpisode(s.NextEpisodeToAir),
		LastEpisode:      mapEpisode(s.LastEpisodeToAir),
	}
	if s.Genres != nil {
		for _, g := range *s.Genres {
			show.Genres = append(show.Genres, provider.Genre{TMDBID: val(g.Id), Name: val(g.Name)})
		}
	}
	if s.Seasons != nil {
		for _, sn := range *s.Seasons {
			show.Seasons = append(show.Seasons, provider.Season{
				TMDBID:       sn.Id,
				SeasonNumber: val(sn.SeasonNumber),
				Name:         val(sn.Name),
				Overview:     val(sn.Overview),
				AirDate:      parseDate(sn.AirDate),
				EpisodeCount: val(sn.EpisodeCount),
				PosterURL:    imageURL(sn.PosterPath),
			})
		}
	}
	return show, nil
}

// GetSeason fetches a season with its episodes.
func (p *Provider) GetSeason(ctx context.Context, tmdbID int64, seasonNumber int) (*provider.Season, error) {
	resp, err := p.client.GetTvSeasonWithResponse(ctx, tmdbID, seasonNumber, p.editor)
	if err != nil {
		return nil, fmt.Errorf("tmdb: get season: %w", safeRequestError(err))
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("tmdb: get season: unexpected status %d", resp.StatusCode())
	}
	d := resp.JSON200
	season := &provider.Season{
		TMDBID:       d.Id,
		SeasonNumber: val(d.SeasonNumber),
		Name:         val(d.Name),
		Overview:     val(d.Overview),
		AirDate:      parseDate(d.AirDate),
		PosterURL:    imageURL(d.PosterPath),
	}
	if d.Episodes != nil {
		for _, e := range *d.Episodes {
			if ep := mapEpisode(&e); ep != nil {
				season.Episodes = append(season.Episodes, *ep)
			}
		}
	}
	season.EpisodeCount = len(season.Episodes)
	return season, nil
}

func mapEpisode(e *tmdbclient.TvEpisode) *provider.Episode {
	if e == nil {
		return nil
	}
	return &provider.Episode{
		TMDBID:        e.Id,
		SeasonNumber:  val(e.SeasonNumber),
		EpisodeNumber: val(e.EpisodeNumber),
		Name:          val(e.Name),
		Overview:      val(e.Overview),
		AirDate:       parseDate(e.AirDate),
		Runtime:       e.Runtime,
		StillURL:      imageURL(e.StillPath),
	}
}

func imageURL(path *string) string {
	if path == nil || *path == "" {
		return ""
	}
	return imageBaseURL + *path
}

func parseDate(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	return &t
}

// val dereferences p, returning the zero value if nil.
func val[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// HTTP transport errors include the request URL, which contains the API key.
func safeRequestError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s: %w", urlErr.Op, urlErr.Err)
	}
	return err
}
