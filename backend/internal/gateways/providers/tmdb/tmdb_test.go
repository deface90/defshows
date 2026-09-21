package tmdb_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deface90/defshows/backend/internal/gateways/providers/tmdb"
)

const searchTvJSON = `{
  "page": 1,
  "results": [
    {"id": 1399, "name": "Game of Thrones", "original_name": "Game of Thrones",
     "overview": "Nine noble families...", "poster_path": "/poster.jpg",
     "first_air_date": "2011-04-17", "popularity": 123.4}
  ]
}`

const tvShowJSON = `{
  "id": 1399, "name": "Game of Thrones", "original_name": "Game of Thrones",
  "overview": "Seven kingdoms", "poster_path": "/p.jpg", "backdrop_path": "/b.jpg",
  "status": "Ended", "in_production": false,
  "first_air_date": "2011-04-17", "last_air_date": "2019-05-19",
  "original_language": "en", "popularity": 500.5,
  "genres": [{"id": 18, "name": "Drama"}, {"id": 10765, "name": "Sci-Fi & Fantasy"}],
  "seasons": [
    {"id": 3624, "season_number": 1, "name": "Season 1", "air_date": "2011-04-17", "episode_count": 10, "poster_path": "/s1.jpg"},
    {"id": 3625, "season_number": 2, "name": "Season 2", "air_date": "2012-04-01", "episode_count": 10}
  ],
  "next_episode_to_air": null,
  "last_episode_to_air": {"id": 999, "season_number": 8, "episode_number": 6, "name": "The Iron Throne", "air_date": "2019-05-19", "runtime": 80}
}`

const seasonJSON = `{
  "id": 3624, "season_number": 1, "name": "Season 1", "air_date": "2011-04-17", "poster_path": "/s1.jpg", "vote_average": 9,
  "episodes": [
    {"id": 63056, "season_number": 1, "episode_number": 1, "name": "Winter Is Coming", "air_date": "2011-04-17", "runtime": 62, "still_path": "/e1.jpg", "vote_average": 8.5, "vote_count": 123},
    {"id": 63057, "season_number": 1, "episode_number": 2, "name": "The Kingsroad", "air_date": "2011-04-24", "runtime": 56}
  ]
}`

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify api_key/language are injected.
		if r.URL.Query().Get("api_key") == "" {
			t.Errorf("missing api_key on %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/search/tv":
			_, _ = w.Write([]byte(searchTvJSON))
		case strings.HasSuffix(r.URL.Path, "/season/1"):
			_, _ = w.Write([]byte(seasonJSON))
		case r.URL.Path == "/tv/1399":
			_, _ = w.Write([]byte(tvShowJSON))
		default:
			http.NotFound(w, r)
		}
	}))
	_ = mux
	t.Cleanup(srv.Close)
	return srv
}

func TestProvider_SearchShows(t *testing.T) {
	srv := newServer(t)
	p, err := tmdb.New(srv.URL, "test-key", "ru-RU", srv.Client())
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	res, err := p.SearchShows(t.Context(), "game of thrones")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("want 1 result, got %d", len(res))
	}
	got := res[0]
	if got.TMDBID != 1399 || got.Title != "Game of Thrones" {
		t.Fatalf("unexpected: %+v", got)
	}
	if got.PosterURL != "https://image.tmdb.org/t/p/w500/poster.jpg" {
		t.Fatalf("poster url: %q", got.PosterURL)
	}
	if got.FirstAirDate == nil || got.FirstAirDate.Year() != 2011 {
		t.Fatalf("first air date: %v", got.FirstAirDate)
	}
}

func TestProvider_GetShow(t *testing.T) {
	srv := newServer(t)
	p, _ := tmdb.New(srv.URL, "k", "ru-RU", srv.Client())

	show, err := p.GetShow(t.Context(), 1399)
	if err != nil {
		t.Fatalf("get show: %v", err)
	}
	if show.Status != "Ended" || show.InProduction {
		t.Fatalf("status/production: %+v", show)
	}
	if len(show.Genres) != 2 || len(show.Seasons) != 2 {
		t.Fatalf("genres=%d seasons=%d", len(show.Genres), len(show.Seasons))
	}
	if show.NextEpisode != nil {
		t.Fatalf("expected nil next episode, got %+v", show.NextEpisode)
	}
	if show.LastEpisode == nil || show.LastEpisode.EpisodeNumber != 6 {
		t.Fatalf("last episode: %+v", show.LastEpisode)
	}
	if show.BackdropURL == "" {
		t.Fatal("expected backdrop url")
	}
}

func TestProvider_GetSeason(t *testing.T) {
	srv := newServer(t)
	p, _ := tmdb.New(srv.URL, "k", "ru-RU", srv.Client())

	season, err := p.GetSeason(t.Context(), 1399, 1)
	if err != nil {
		t.Fatalf("get season: %v", err)
	}
	if season.SeasonNumber != 1 || len(season.Episodes) != 2 {
		t.Fatalf("unexpected season: %+v", season)
	}
	if season.EpisodeCount != 2 {
		t.Fatalf("episode count: %d", season.EpisodeCount)
	}
	if season.VoteAverage != 9 {
		t.Fatalf("season rating: %v", season.VoteAverage)
	}
	if season.Episodes[0].VoteAverage != 8.5 || season.Episodes[0].VoteCount != 123 {
		t.Fatalf("episode rating: %+v", season.Episodes[0])
	}
	if season.Episodes[1].VoteAverage != 0 || season.Episodes[1].VoteCount != 0 {
		t.Fatal("missing rating should remain zero")
	}
	e0 := season.Episodes[0]
	if e0.Name != "Winter Is Coming" || e0.Runtime == nil || *e0.Runtime != 62 {
		t.Fatalf("episode 0: %+v", e0)
	}
	if e0.StillURL != "https://image.tmdb.org/t/p/w500/e1.jpg" {
		t.Fatalf("still url: %q", e0.StillURL)
	}
}

// A transport failure must remain diagnosable without logging the API key.
func TestSearchTransportErrorOmitsCredentials(t *testing.T) {
	client := &http.Client{Transport: failingTransport{}}
	p, err := tmdb.New("https://example.invalid/3", "secret-api-key", "ru-RU", client)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.SearchShows(t.Context(), "test")
	if err == nil || !strings.Contains(err.Error(), "connection failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "secret-api-key") || strings.Contains(err.Error(), "api_key") {
		t.Fatalf("credentials in error: %v", err)
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("connection failed")
}
