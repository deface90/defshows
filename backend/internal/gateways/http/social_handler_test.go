package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/labstack/echo/v4"

	authapi "github.com/deface90/defshows/backend/pkg/server/auth"
	socialapi "github.com/deface90/defshows/backend/pkg/server/social"
	usersapi "github.com/deface90/defshows/backend/pkg/server/users"
)

// registerUser registers an account and returns its id + access token.
func registerUser(t *testing.T, e *echo.Echo, email string) (int64, string) {
	t.Helper()
	rec := doJSON(t, e, http.MethodPost, "/auth/register", "", map[string]string{"email": email, "password": "pw12345"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s: %d (%s)", email, rec.Code, rec.Body.String())
	}
	var reg authapi.AuthResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)
	return reg.User.Id, reg.Tokens.AccessToken
}

func TestSocial_FollowFlow(t *testing.T) {
	e := newWebServer(t)

	aID, aTok := registerUser(t, e, "alice@soc.com")
	bID, bTok := registerUser(t, e, "bob@soc.com")
	_ = aID

	// Make alice public.
	if rec := doJSON(t, e, http.MethodPatch, "/me/settings", aTok, map[string]any{"is_public": true}); rec.Code != http.StatusOK {
		t.Fatalf("alice public: %d", rec.Code)
	}

	// bob follows alice (public) → accepted immediately.
	rec := doJSON(t, e, http.MethodPost, "/me/follows/"+strconv.FormatInt(aID, 10), bTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("follow public: %d (%s)", rec.Code, rec.Body.String())
	}
	var res socialapi.FollowResult
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Status != socialapi.Accepted {
		t.Fatalf("want accepted, got %s", res.Status)
	}

	// Self-follow → 400.
	if rec := doJSON(t, e, http.MethodPost, "/me/follows/"+strconv.FormatInt(bID, 10), bTok, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("self-follow: want 400, got %d", rec.Code)
	}

	// alice follows bob (private) → pending.
	rec = doJSON(t, e, http.MethodPost, "/me/follows/"+strconv.FormatInt(bID, 10), aTok, nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if rec.Code != http.StatusOK || res.Status != socialapi.Pending {
		t.Fatalf("follow private: %d %s", rec.Code, res.Status)
	}

	// bob sees one incoming request (alice).
	rec = doJSON(t, e, http.MethodGet, "/me/follows/incoming", bTok, nil)
	var incoming socialapi.FollowUserList
	_ = json.Unmarshal(rec.Body.Bytes(), &incoming)
	if incoming.Total != 1 || len(incoming.Users) != 1 || incoming.Users[0].Id != aID {
		t.Fatalf("incoming: %+v", incoming)
	}

	// Before approval, alice cannot see bob's (private) collection → 403.
	if rec := doJSON(t, e, http.MethodGet, "/users/"+strconv.FormatInt(bID, 10)+"/shows", aTok, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("private shows before approval: want 403, got %d", rec.Code)
	}

	// bob approves alice.
	if rec := doJSON(t, e, http.MethodPost, "/me/follows/incoming/"+strconv.FormatInt(aID, 10)+"/approve", bTok, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("approve: %d", rec.Code)
	}
	// Now alice (accepted follower) can see bob's collection → 200.
	if rec := doJSON(t, e, http.MethodGet, "/users/"+strconv.FormatInt(bID, 10)+"/shows", aTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("private shows after approval: want 200, got %d", rec.Code)
	}

	// bob's profile (seen by alice) reports is_following=accepted + 1 follower.
	rec = doJSON(t, e, http.MethodGet, "/users/"+strconv.FormatInt(bID, 10), aTok, nil)
	var prof usersapi.UserProfile
	_ = json.Unmarshal(rec.Body.Bytes(), &prof)
	if prof.IsFollowing != usersapi.Accepted || prof.FollowersCount != 1 {
		t.Fatalf("profile: following=%s followers=%d", prof.IsFollowing, prof.FollowersCount)
	}

	// Approving a non-existent request → 404.
	if rec := doJSON(t, e, http.MethodPost, "/me/follows/incoming/"+strconv.FormatInt(aID, 10)+"/approve", aTok, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("approve missing: want 404, got %d", rec.Code)
	}

	// alice unfollows bob; her follow state resets to none.
	if rec := doJSON(t, e, http.MethodDelete, "/me/follows/"+strconv.FormatInt(bID, 10), aTok, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("unfollow: %d", rec.Code)
	}
	rec = doJSON(t, e, http.MethodGet, "/users/"+strconv.FormatInt(bID, 10), aTok, nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &prof)
	if prof.IsFollowing != usersapi.None {
		t.Fatalf("after unfollow: following=%s", prof.IsFollowing)
	}
}

func TestSocial_Feeds(t *testing.T) {
	e := newWebServer(t)

	aID, aTok := registerUser(t, e, "feedalice@soc.com")
	_, bTok := registerUser(t, e, "feedbob@soc.com")
	cID, cTok := registerUser(t, e, "feedcarol@soc.com")

	// Alice is public; carol stays private.
	if rec := doJSON(t, e, http.MethodPatch, "/me/settings", aTok, map[string]any{"is_public": true}); rec.Code != http.StatusOK {
		t.Fatalf("alice public: %d", rec.Code)
	}

	// Alice adds and rates a show → added_show + rated_show events.
	rec := doJSON(t, e, http.MethodPost, "/me/shows", aTok, map[string]int64{"tmdb_id": 1399})
	if rec.Code != http.StatusCreated {
		t.Fatalf("alice add show: %d (%s)", rec.Code, rec.Body.String())
	}
	var added struct {
		ShowID int64 `json:"show_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &added)
	if rec := doJSON(t, e, http.MethodPatch, "/me/shows/"+strconv.FormatInt(added.ShowID, 10), aTok, map[string]any{"rating": 8}); rec.Code != http.StatusOK {
		t.Fatalf("alice rate: %d (%s)", rec.Code, rec.Body.String())
	}

	// Carol (private, unfollowed) adds a show — must never leak into bob's home feed.
	if rec := doJSON(t, e, http.MethodPost, "/me/shows", cTok, map[string]int64{"tmdb_id": 1399}); rec.Code != http.StatusCreated {
		t.Fatalf("carol add show: %d", rec.Code)
	}

	// Bob follows alice (public → accepted).
	if rec := doJSON(t, e, http.MethodPost, "/me/follows/"+strconv.FormatInt(aID, 10), bTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("bob follow alice: %d", rec.Code)
	}

	// Home feed: bob sees only alice's events (added_show + rated_show), with actor set.
	rec = doJSON(t, e, http.MethodGet, "/me/feed", bTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("home feed: %d (%s)", rec.Code, rec.Body.String())
	}
	var home socialapi.FeedPage
	_ = json.Unmarshal(rec.Body.Bytes(), &home)
	if len(home.Cards) != 2 {
		t.Fatalf("home feed should have alice's 2 cards, got %d: %s", len(home.Cards), rec.Body.String())
	}
	var sawRating bool
	for _, card := range home.Cards {
		if card.Actor == nil || card.Actor.Id != aID {
			t.Fatalf("home card missing alice as actor: %+v", card)
		}
		if card.Type == socialapi.FeedCardType("rated_show") {
			if card.Rating == nil || *card.Rating != 8 {
				t.Fatalf("rated_show card should carry rating 8, got %+v", card.Rating)
			}
			sawRating = true
		}
	}
	if !sawRating {
		t.Fatal("home feed missing the rated_show card")
	}

	// Profile feed of a public user → 200; of a private non-followed user → 403.
	if rec := doJSON(t, e, http.MethodGet, "/users/"+strconv.FormatInt(aID, 10)+"/feed", bTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("alice profile feed: want 200, got %d", rec.Code)
	}
	if rec := doJSON(t, e, http.MethodGet, "/users/"+strconv.FormatInt(cID, 10)+"/feed", bTok, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("carol private profile feed: want 403, got %d", rec.Code)
	}

	// Cursor round-trip: one card per page, second page continues past the first.
	rec = doJSON(t, e, http.MethodGet, "/users/"+strconv.FormatInt(aID, 10)+"/feed?limit=1", aTok, nil)
	var p1 socialapi.FeedPage
	_ = json.Unmarshal(rec.Body.Bytes(), &p1)
	if len(p1.Cards) != 1 || p1.NextCursor == nil || *p1.NextCursor == "" {
		t.Fatalf("page1: cards=%d next=%v", len(p1.Cards), p1.NextCursor)
	}
	rec = doJSON(t, e, http.MethodGet, "/users/"+strconv.FormatInt(aID, 10)+"/feed?limit=1&cursor="+*p1.NextCursor, aTok, nil)
	var p2 socialapi.FeedPage
	_ = json.Unmarshal(rec.Body.Bytes(), &p2)
	if len(p2.Cards) != 1 {
		t.Fatalf("page2: cards=%d", len(p2.Cards))
	}
	if p1.Cards[0].Type == p2.Cards[0].Type {
		t.Fatalf("page2 should differ from page1 (rated_show then added_show), both %s", p1.Cards[0].Type)
	}
}

func TestSocial_AutoAcceptOnGoingPublic(t *testing.T) {
	e := newWebServer(t)
	ownerID, ownerTok := registerUser(t, e, "owner@soc.com")
	_, fanTok := registerUser(t, e, "fan@soc.com")

	// Owner is private by default; fan's follow is pending.
	rec := doJSON(t, e, http.MethodPost, "/me/follows/"+strconv.FormatInt(ownerID, 10), fanTok, nil)
	var res socialapi.FollowResult
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Status != socialapi.Pending {
		t.Fatalf("want pending, got %s", res.Status)
	}

	// Owner switches to public → the pending request is auto-accepted.
	if rec := doJSON(t, e, http.MethodPatch, "/me/settings", ownerTok, map[string]any{"is_public": true}); rec.Code != http.StatusOK {
		t.Fatalf("go public: %d", rec.Code)
	}
	rec = doJSON(t, e, http.MethodGet, "/me/follows/incoming", ownerTok, nil)
	var incoming socialapi.FollowUserList
	_ = json.Unmarshal(rec.Body.Bytes(), &incoming)
	if incoming.Total != 0 {
		t.Fatalf("pending after going public: %d", incoming.Total)
	}
}
