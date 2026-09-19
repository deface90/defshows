package tmdb_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deface90/defshows/backend/internal/gateways/providers/tmdb"
)

type referenceTransport func(*http.Request) (*http.Response, error)

func (f referenceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProvider_ReferenceLinks(t *testing.T) {
	ru := "https://ru.wikipedia.org/wiki/Game_of_Thrones"
	en := "https://en.wikipedia.org/wiki/Game_of_Thrones"
	for _, tc := range []struct {
		name, external, wikiBody, wantWiki string
		status                             int
		wantIMDb, wantWikiNil, wantRequest bool
	}{
		{name: "Russian preferred", external: `{"imdb_id":"tt0944947","wikidata_id":"Q23572"}`, wikiBody: fmt.Sprintf(`{"entities":{"Q23572":{"sitelinks":{"ruwiki":{"url":%q},"enwiki":{"url":%q}}}}}`, ru, en), wantWiki: ru, wantIMDb: true, wantRequest: true},
		{name: "English fallback", external: `{"imdb_id":"tt0944947","wikidata_id":"Q23572"}`, wikiBody: fmt.Sprintf(`{"entities":{"Q23572":{"sitelinks":{"enwiki":{"url":%q}}}}}`, en), wantWiki: en, wantIMDb: true, wantRequest: true},
		{name: "no article", external: `{"wikidata_id":"Q23572"}`, wikiBody: `{"entities":{"Q23572":{"sitelinks":{}}}}`, wantRequest: true},
		{name: "null IDs", external: `{"imdb_id":null,"wikidata_id":null}`},
		{name: "invalid IDs", external: `{"imdb_id":"tt123/evil","wikidata_id":"Q123/evil"}`},
		{name: "unsafe article URL", external: `{"wikidata_id":"Q23572"}`, wikiBody: `{"entities":{"Q23572":{"sitelinks":{"ruwiki":{"url":"https://evil.example/wiki/test"}}}}}`, wantRequest: true},
		{name: "Wikipedia unavailable", external: `{"imdb_id":"tt0944947","wikidata_id":"Q23572"}`, status: 503, wantIMDb: true, wantWikiNil: true, wantRequest: true},
		{name: "bad JSON", external: `{"wikidata_id":"Q23572"}`, wikiBody: `{`, wantWikiNil: true, wantRequest: true},
		{name: "API error", external: `{"wikidata_id":"Q23572"}`, wikiBody: `{"error":{"code":"maxlag"}}`, wantWikiNil: true, wantRequest: true},
		{name: "network error", external: `{"wikidata_id":"Q23572"}`, status: -1, wantWikiNil: true, wantRequest: true},
		{name: "no external IDs", external: `null`, wantWikiNil: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wikiRequests := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("append_to_response") != "external_ids" {
					t.Error("external IDs not requested")
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"id":1399,"name":"Game of Thrones","external_ids":%s}`, tc.external)
			}))
			defer srv.Close()
			client := &http.Client{Transport: referenceTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host != "www.wikidata.org" {
					return srv.Client().Transport.RoundTrip(r)
				}
				wikiRequests++
				q := r.URL.Query()
				if q.Get("api_key") != "" {
					t.Error("TMDB key leaked to Wikidata")
				}
				if q.Get("ids") != "Q23572" || q.Get("props") != "sitelinks/urls" || q.Get("sitefilter") != "ruwiki|enwiki" {
					t.Errorf("unexpected query: %v", q)
				}
				if r.Header.Get("User-Agent") == "" {
					t.Error("missing User-Agent")
				}
				if tc.status == -1 {
					return nil, fmt.Errorf("connection failed")
				}
				status := tc.status
				if status == 0 {
					status = 200
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(tc.wikiBody)), Header: make(http.Header)}, nil
			})}
			p, err := tmdb.New(srv.URL, "secret", "ru-RU", client)
			if err != nil {
				t.Fatal(err)
			}
			show, err := p.GetShow(t.Context(), 1399)
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantIMDb && (show.IMDbURL == nil || *show.IMDbURL != "https://www.imdb.com/title/tt0944947/") {
				t.Fatalf("IMDb: %v", show.IMDbURL)
			}
			if !tc.wantIMDb && show.IMDbURL != nil && *show.IMDbURL != "" {
				t.Fatalf("unexpected IMDb: %s", *show.IMDbURL)
			}
			if (show.WikipediaURL == nil) != tc.wantWikiNil {
				t.Fatalf("Wikipedia nil=%v, want %v", show.WikipediaURL == nil, tc.wantWikiNil)
			}
			if show.WikipediaURL != nil && *show.WikipediaURL != tc.wantWiki {
				t.Fatalf("Wikipedia=%s, want %s", *show.WikipediaURL, tc.wantWiki)
			}
			if (wikiRequests > 0) != tc.wantRequest {
				t.Fatalf("Wikipedia requests: %d", wikiRequests)
			}
		})
	}
}
