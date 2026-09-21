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

	for _, tc := range []struct {
		path  string
		total int64
		count int
		name  string
	}{
		{"/users?page=1&page_size=1", 2, 1, "alice"},
		{"/users?page=2&page_size=1", 2, 1, "bob"},
		{"/users?q=ALIce&page_size=1", 1, 1, "alice"},
		{"/users?page=3&page_size=1", 2, 0, ""},
		{"/users?q=missing", 0, 0, ""},
	} {
		response := doJSON(t, e, http.MethodGet, tc.path, alice.Tokens.AccessToken, nil)
		if response.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", tc.path, response.Code, response.Body.String())
		}
		var page usersapi.UserList
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if page.Total != tc.total || len(page.Users) != tc.count {
			t.Fatalf("%s: %+v", tc.path, page)
		}
		if tc.count > 0 && page.Users[0].DisplayName != tc.name {
			t.Fatalf("%s: wrong user %+v", tc.path, page.Users)
		}
	}
	for _, path := range []string{"/users?page=0", "/users?page=-1", "/users?page=1000001", "/users?page_size=101", "/users?page_size=0", "/users?page=abc"} {
		response := doJSON(t, e, http.MethodGet, path, alice.Tokens.AccessToken, nil)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: got %d", path, response.Code)
		}
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
