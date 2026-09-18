package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	authapi "github.com/deface90/defshows/backend/pkg/server/auth"
	notificationsapi "github.com/deface90/defshows/backend/pkg/server/notifications"
)

func TestNotificationsHandler_PrefsAndLink(t *testing.T) {
	e := newWebServer(t)

	rec := doJSON(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"email": "notif@example.com", "password": "pw12345",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d", rec.Code)
	}
	var reg authapi.AuthResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)
	token := reg.Tokens.AccessToken

	// Default prefs.
	rec = doJSON(t, e, http.MethodGet, "/me/notifications/prefs", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get prefs: %d", rec.Code)
	}
	var prefs notificationsapi.Prefs
	_ = json.Unmarshal(rec.Body.Bytes(), &prefs)
	if !prefs.EpisodeRelease {
		t.Fatalf("expected default episode_release=true: %+v", prefs)
	}

	// Update prefs.
	rec = doJSON(t, e, http.MethodPatch, "/me/notifications/prefs", token,
		map[string]any{"episode_release": false, "lead_time_hours": 48})
	if rec.Code != http.StatusOK {
		t.Fatalf("patch prefs: %d", rec.Code)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &prefs)
	if prefs.EpisodeRelease || prefs.LeadTimeHours != 48 {
		t.Fatalf("prefs not updated: %+v", prefs)
	}

	// Telegram link.
	rec = doJSON(t, e, http.MethodGet, "/me/telegram/link", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("telegram link: %d", rec.Code)
	}
	var link notificationsapi.TelegramLink
	_ = json.Unmarshal(rec.Body.Bytes(), &link)
	if !strings.HasPrefix(link.Url, "https://t.me/defShowsBot?start=") {
		t.Fatalf("unexpected link: %s", link.Url)
	}

	// Feed (empty).
	rec = doJSON(t, e, http.MethodGet, "/me/notifications", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("feed: %d", rec.Code)
	}
}
