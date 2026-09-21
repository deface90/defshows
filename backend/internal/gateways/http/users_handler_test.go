package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	authapi "github.com/deface90/defshows/backend/pkg/server/auth"
	usersapi "github.com/deface90/defshows/backend/pkg/server/users"
)

// TestUsers_DirectoryAndVisibility exercises the users directory and the
// public/private gate on a user's tracked-shows list.
func TestUsers_DirectoryAndVisibility(t *testing.T) {
	e := newWebServer(t)

	register := func(email string) authapi.AuthResponse {
		rec := doJSON(t, e, http.MethodPost, "/auth/register", "", map[string]string{
			"email": email, "password": "pw12345",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("register %s: %d (%s)", email, rec.Code, rec.Body.String())
		}
		var out authapi.AuthResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return out
	}

	alice := register("alice@example.com")
	bob := register("bob@example.com")
	bobShows := "/users/" + strconv.FormatInt(bob.User.Id, 10) + "/shows"

	// Bob tracks a show.
	if rec := doJSON(t, e, http.MethodPost, "/me/shows", bob.Tokens.AccessToken,
		map[string]int64{"tmdb_id": 1399}); rec.Code != http.StatusCreated {
		t.Fatalf("bob add show: %d (%s)", rec.Code, rec.Body.String())
	}

	// Directory lists both users.
	rec := doJSON(t, e, http.MethodGet, "/users", alice.Tokens.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list users: %d", rec.Code)
	}
	var list usersapi.UserList
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Users) != 2 {
		t.Fatalf("want 2 users, got %d", len(list.Users))
	}

	// Bob is private by default → Alice is forbidden.
	rec = doJSON(t, e, http.MethodGet, bobShows, alice.Tokens.AccessToken, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("private profile: want 403, got %d", rec.Code)
	}

	// Bob can always see his own shows.
	rec = doJSON(t, e, http.MethodGet, bobShows, bob.Tokens.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner view: want 200, got %d", rec.Code)
	}

	// Bob goes public.
	if rec := doJSON(t, e, http.MethodPatch, "/me/settings", bob.Tokens.AccessToken,
		map[string]any{"timezone": "UTC", "is_public": true}); rec.Code != http.StatusOK {
		t.Fatalf("bob set public: %d (%s)", rec.Code, rec.Body.String())
	}

	// Now Alice sees Bob's tracked shows.
	rec = doJSON(t, e, http.MethodGet, bobShows, alice.Tokens.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("public profile: want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var tracked usersapi.TrackedShowList
	_ = json.Unmarshal(rec.Body.Bytes(), &tracked)
	if len(tracked.Tracked) != 1 {
		t.Fatalf("want 1 tracked show, got %d", len(tracked.Tracked))
	}

	if tracked.Tracked[0].Show.SeasonCount == nil || *tracked.Tracked[0].Show.SeasonCount != 1 {
		t.Fatalf("season count missing from public profile: %+v", tracked.Tracked[0].Show)
	}

	// Profile metadata reflects the public flag and shows count.
	rec = doJSON(t, e, http.MethodGet, "/users/"+strconv.FormatInt(bob.User.Id, 10), alice.Tokens.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get profile: %d", rec.Code)
	}
	var profile usersapi.UserProfile
	_ = json.Unmarshal(rec.Body.Bytes(), &profile)
	if !profile.IsPublic || profile.ShowsCount != 1 {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}
