package tmdb_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deface90/defshows/backend/internal/gateways/providers/tmdb"
	"github.com/deface90/defshows/backend/pkg/provider"
)

func TestDiscoverFiltersAndPagination(t *testing.T) {
	for _, genres := range []string{"18|35", "18,35"} {
		t.Run(genres, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/3/discover/tv" {
					t.Errorf("path: %s", r.URL.Path)
				}
				for key, want := range map[string]string{"api_key": "test-key", "language": "ru-RU", "with_origin_country": "US", "with_genres": genres, "vote_average.gte": "7.3", "vote_average.lte": "9", "vote_count.gte": "200", "first_air_date.gte": "2000-01-01", "first_air_date.lte": "2020-12-31", "page": "2", "sort_by": "vote_average.desc", "include_adult": "false"} {
					if got := r.URL.Query().Get(key); got != want {
						t.Errorf("%s: got %q want %q", key, got, want)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"page":2,"total_pages":800,"total_results":16000,"results":[{"id":123,"name":"Test","vote_average":8.5,"vote_count":400}]}`))
			}))
			defer srv.Close()
			p, err := tmdb.New(srv.URL+"/3", "test-key", "ru-RU", srv.Client())
			if err != nil {
				t.Fatal(err)
			}
			page, err := p.Discover(t.Context(), provider.DiscoverOptions{Country: "US", Genres: genres, RatingMin: 7.3, RatingMax: 9, VotesMin: 200, DateFrom: "2000-01-01", DateTo: "2020-12-31", Page: 2, Sort: "vote_average.desc"})
			if err != nil {
				t.Fatal(err)
			}
			if page.Page != 2 || page.TotalPages != 500 || page.TotalResults != 16000 || len(page.Results) != 1 || page.Results[0].VoteCount != 400 {
				t.Fatalf("page: %+v", page)
			}
		})
	}
}

func TestTrending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/3/trending/tv/week" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("api_key"); got != "test-key" {
			t.Errorf("api_key: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"page":1,"total_pages":1,"total_results":1,"results":[{"id":123,"name":"Test","vote_average":8.5,"vote_count":400}]}`))
	}))
	defer srv.Close()
	p, err := tmdb.New(srv.URL+"/3", "test-key", "ru-RU", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	page, err := p.Trending(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if page.TotalResults != 1 || len(page.Results) != 1 || page.Results[0].TMDBID != 123 {
		t.Fatalf("page: %+v", page)
	}
}

func TestDiscoveryReferencesCached(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("language") != "ru-RU" {
			t.Error("missing language")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/genre/tv/list":
			_, _ = w.Write([]byte(`{"genres":[{"id":18,"name":"Драма"}]}`))
		case "/configuration/countries":
			_, _ = w.Write([]byte(`[{"iso_3166_1":"US","english_name":"United States","native_name":"United States"}]`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	p, _ := tmdb.New(srv.URL, "key", "ru-RU", srv.Client())
	for i := 0; i < 2; i++ {
		refs, err := p.DiscoveryFilters(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(refs.Genres) != 1 || refs.Genres[0].Name != "Драма" || len(refs.Countries) != 1 || refs.Countries[0].Code != "US" {
			t.Fatalf("references: %+v", refs)
		}
	}
	if calls != 2 {
		t.Fatalf("wanted only two upstream requests, got %d", calls)
	}
}

func TestDiscoveryReferencesFailureNotCached(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	p, _ := tmdb.New(srv.URL, "key", "ru-RU", srv.Client())
	for i := 0; i < 2; i++ {
		if _, err := p.DiscoveryFilters(t.Context()); err == nil {
			t.Fatal("expected error")
		}
	}
	if calls != 2 {
		t.Fatalf("failed response cached: %d calls", calls)
	}
	if _, err := p.Discover(t.Context(), provider.DiscoverOptions{}); err == nil {
		t.Fatal("expected discover error")
	}
}
