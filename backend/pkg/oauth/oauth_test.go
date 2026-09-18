package oauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deface90/defshows/backend/pkg/oauth"
)

func TestGoogle_AuthURLAndExchange(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer at" {
			t.Errorf("missing bearer token: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"g-123","email":"user@gmail.com","name":"User G"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := oauth.NewGoogle("client-id", "secret", &oauth.GoogleOptions{
		AuthURL:     srv.URL + "/authorize",
		TokenURL:    srv.URL + "/token",
		UserInfoURL: srv.URL + "/userinfo",
		HTTPClient:  srv.Client(),
	})

	if p.Name() != "google" {
		t.Fatalf("name: %s", p.Name())
	}

	authURL := p.AuthURL("state123", "https://app/cb")
	if !strings.Contains(authURL, "state123") || !strings.Contains(authURL, "client-id") {
		t.Fatalf("auth url missing params: %s", authURL)
	}

	info, err := p.Exchange(context.Background(), "the-code", "https://app/cb")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if info.ProviderUserID != "g-123" || info.Email != "user@gmail.com" || info.Name != "User G" {
		t.Fatalf("unexpected userinfo: %+v", info)
	}
}
