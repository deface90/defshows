package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/pkg/auth"
)

type imageTransport func(*http.Request) (*http.Response, error)

func (f imageTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTMDBImagesPublicProxy(t *testing.T) {
	for _, tc := range []struct {
		name, path, contentType, body string
		upstreamStatus, wantStatus    int
		length                        int64
		networkError                  bool
	}{
		{name: "image without authentication or S3", path: "/images/tmdb/w500/poster.jpg?url=https://evil.example", contentType: "image/jpeg", body: "image bytes", upstreamStatus: 200, wantStatus: 200},
		{name: "not modified", path: "/images/tmdb/w500/poster.jpg", upstreamStatus: 304, wantStatus: 304},
		{name: "missing", path: "/images/tmdb/w500/poster.jpg", upstreamStatus: 404, wantStatus: 404},
		{name: "upstream error", path: "/images/tmdb/w500/poster.jpg", upstreamStatus: 503, wantStatus: 502},
		{name: "network error", path: "/images/tmdb/w500/poster.jpg", networkError: true, wantStatus: 502},
		{name: "redirect blocked", path: "/images/tmdb/w500/poster.jpg", upstreamStatus: 302, wantStatus: 502},
		{name: "HTML blocked", path: "/images/tmdb/w500/poster.jpg", upstreamStatus: 200, contentType: "text/html", body: "<html>error</html>", wantStatus: 502},
		{name: "empty body", path: "/images/tmdb/w500/poster.jpg", upstreamStatus: 200, contentType: "image/jpeg", wantStatus: 502},
		{name: "oversize declared", path: "/images/tmdb/w500/poster.jpg", upstreamStatus: 200, contentType: "image/jpeg", body: "x", length: maxTMDBImageBytes + 1, wantStatus: 502},
		{name: "oversize chunked", path: "/images/tmdb/w500/poster.jpg", upstreamStatus: 200, contentType: "image/jpeg", body: strings.Repeat("x", maxTMDBImageBytes+1), length: -1, wantStatus: 502},
		{name: "unsupported size", path: "/images/tmdb/w99999/poster.jpg", wantStatus: 400},
		{name: "path traversal", path: "/images/tmdb/w500/..%2Fsecret.jpg", wantStatus: 400},
		{name: "unsupported extension", path: "/images/tmdb/w500/test.svg", wantStatus: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: imageTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.String() != "https://image.tmdb.org/t/p/w500/poster.jpg" {
					t.Fatalf("unexpected upstream: %s", r.URL)
				}
				if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
					t.Fatal("credentials forwarded")
				}
				if r.Header.Get("If-None-Match") != `"poster-etag"` {
					t.Fatal("missing conditional header")
				}
				if tc.networkError {
					return nil, fmt.Errorf("proxy unavailable")
				}
				return &http.Response{
					StatusCode: tc.upstreamStatus, ContentLength: tc.length,
					Body:   io.NopCloser(strings.NewReader(tc.body)),
					Header: http.Header{"Content-Type": {tc.contentType}, "Etag": {`"poster-etag"`}, "Location": {"https://evil.example/"}},
				}, nil
			})}
			e := NewWebRouter(nil, nil, nil, nil, nil, nil, nil, auth.NewJWTManager("secret", time.Hour), nil, client)
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Cookie", "private=value")
			req.Header.Set("If-None-Match", `"poster-etag"`)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus == 400 && calls != 0 {
				t.Fatalf("invalid path fetched: %d calls", calls)
			}
			if tc.wantStatus != 400 && calls != 1 {
				t.Fatalf("want one request, got %d", calls)
			}
			if tc.wantStatus == 200 || tc.wantStatus == 304 {
				if rec.Header().Get("Cache-Control") != "public, max-age=86400" {
					t.Fatal("missing cache policy")
				}
				if rec.Header().Get("ETag") != `"poster-etag"` {
					t.Fatal("missing ETag")
				}
			} else if rec.Header().Get("Cache-Control") != "" {
				t.Fatal("error cached")
			}
			if tc.wantStatus == 200 && rec.Body.String() != tc.body {
				t.Fatal("image bytes changed")
			}
			if tc.wantStatus == 304 && rec.Body.Len() != 0 {
				t.Fatal("304 has body")
			}
		})
	}
}
