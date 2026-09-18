package omdb_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deface90/defshows/backend/internal/gateways/providers/omdb"
)

const foundJSON = `{
  "Title": "Game of Thrones", "Year": "2011", "Response": "True",
  "imdbRating": "9.2", "imdbVotes": "2,145,000", "Metascore": "86",
  "Ratings": [
    {"Source": "Internet Movie Database", "Value": "9.2/10"},
    {"Source": "Rotten Tomatoes", "Value": "89%"},
    {"Source": "Metacritic", "Value": "86/100"}
  ]
}`

const notFoundJSON = `{"Response": "False", "Error": "Series not found!"}`

func serve(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("apikey") == "" {
			t.Errorf("missing apikey")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestProvider_Ratings(t *testing.T) {
	srv := serve(t, foundJSON)
	p, err := omdb.New(srv.URL, "key", srv.Client())
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ratings, err := p.Ratings(t.Context(), "Game of Thrones", 2011)
	if err != nil {
		t.Fatalf("ratings: %v", err)
	}
	if len(ratings) != 3 {
		t.Fatalf("want 3 ratings, got %d: %+v", len(ratings), ratings)
	}

	bySource := map[string]string{}
	var imdbVotes *int64
	for _, r := range ratings {
		bySource[r.Source] = r.Value
		if r.Source == "imdb" {
			imdbVotes = r.Votes
		}
	}
	if bySource["imdb"] != "9.2/10" || bySource["rotten_tomatoes"] != "89%" || bySource["metacritic"] != "86/100" {
		t.Fatalf("unexpected mapping: %+v", bySource)
	}
	if imdbVotes == nil || *imdbVotes != 2145000 {
		t.Fatalf("imdb votes: %v", imdbVotes)
	}
}

func TestProvider_Ratings_NotFound(t *testing.T) {
	srv := serve(t, notFoundJSON)
	p, _ := omdb.New(srv.URL, "key", srv.Client())
	ratings, err := p.Ratings(t.Context(), "Nope", 0)
	if err != nil {
		t.Fatalf("expected nil error for not-found, got %v", err)
	}
	if len(ratings) != 0 {
		t.Fatalf("want 0 ratings, got %d", len(ratings))
	}
}
