package oauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deface90/defshows/backend/pkg/oauth"
)

func TestVK_Exchange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok","user_id":12345,"email":"vk@ex.com"}`))
	}))
	defer srv.Close()

	p := oauth.NewVK("cid", "secret", &oauth.VKOptions{TokenURL: srv.URL, HTTPClient: srv.Client()})
	if p.Name() != "vk" {
		t.Fatalf("name: %s", p.Name())
	}
	info, err := p.Exchange(context.Background(), "code", "https://app/cb")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if info.ProviderUserID != "12345" || info.Email != "vk@ex.com" {
		t.Fatalf("unexpected: %+v", info)
	}
}

func TestVK_Exchange_NoEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"tok","user_id":777}`))
	}))
	defer srv.Close()

	p := oauth.NewVK("cid", "secret", &oauth.VKOptions{TokenURL: srv.URL, HTTPClient: srv.Client()})
	info, err := p.Exchange(context.Background(), "code", "cb")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if info.ProviderUserID != "777" || info.Email != "" {
		t.Fatalf("expected empty email, got %+v", info)
	}
}

func TestYandex_Exchange(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/info", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ya-1","default_email":"y@ya.ru","real_name":"Ya User"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := oauth.NewYandex("cid", "secret", &oauth.YandexOptions{
		TokenURL:    srv.URL + "/token",
		UserInfoURL: srv.URL + "/info",
		HTTPClient:  srv.Client(),
	})
	info, err := p.Exchange(context.Background(), "code", "cb")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if info.ProviderUserID != "ya-1" || info.Email != "y@ya.ru" || info.Name != "Ya User" {
		t.Fatalf("unexpected: %+v", info)
	}
}
