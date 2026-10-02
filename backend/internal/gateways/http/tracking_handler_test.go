package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	httpapi "github.com/deface90/defshows/backend/internal/gateways/http"
	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/internal/testutil"
	"github.com/deface90/defshows/backend/pkg/auth"
	"github.com/deface90/defshows/backend/pkg/crud"
	authapi "github.com/deface90/defshows/backend/pkg/server/auth"
	showsapi "github.com/deface90/defshows/backend/pkg/server/shows"
	trackingapi "github.com/deface90/defshows/backend/pkg/server/tracking"
)

func newWebServer(t *testing.T) *echo.Echo {
	t.Helper()
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows", "genres")

	userRepo := repository.NewUserRepository(gdb)
	catalogRepo := repository.NewCatalogRepository(gdb)
	trackingRepo := repository.NewTrackingRepository(gdb)
	socialRepo := repository.NewSocialRepository(gdb)

	notificationRepo := repository.NewNotificationRepository(gdb)
	notesRepo := repository.NewNotesRepository(gdb)

	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	authUC := usecase.NewAuthUsecase(userRepo, jwtMgr, time.Hour)
	catalogUC := usecase.NewCatalogUsecase(catalogRepo, fakeShowProvider{})
	trackingUC := usecase.NewTrackingUsecase(trackingRepo, catalogUC)
	homeUC := usecase.NewHomeUsecase(repository.NewHomeRepository(gdb))
	socialUC := usecase.NewSocialUsecase(socialRepo, userRepo, notificationRepo)
	feedUC := usecase.NewFeedUsecase(repository.NewActivityRepository(gdb), socialRepo, 12*time.Hour)
	notificationUC := usecase.NewNotificationUsecase(notificationRepo, userRepo, "defShowsBot", time.Hour)
	notesUC := usecase.NewNotesUsecase(notesRepo)
	reportUC := usecase.NewReportUsecase(repository.NewReportRepository(gdb), userRepo)
	adminH := httpapi.NewAdminHandler(crud.NewRepository[entity.DubbingStudio](gdb), reportUC, userRepo)
	usersH := httpapi.NewUsersHandler(authUC, trackingUC, socialUC)

	return httpapi.NewWebRouter(
		httpapi.NewAuthHandler(authUC),
		httpapi.NewShowsHandler(catalogUC),
		httpapi.NewTrackingHandler(trackingUC, homeUC, authUC, socialUC),
		httpapi.NewNotificationsHandler(notificationUC),
		httpapi.NewNotesHandler(notesUC),
		adminH,
		usersH,
		httpapi.NewSocialHandler(socialUC, feedUC, reportUC, catalogUC, userRepo),
		jwtMgr,
		nil,
		nil,
	)
}

// TestWebSlice_EndToEnd exercises the MVP vertical slice over HTTP:
// register -> add show (imports) -> list -> watch episode -> progress -> settings.
func TestWebSlice_EndToEnd(t *testing.T) {
	e := newWebServer(t)

	// Register → token.
	rec := doJSON(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"email": "slice@example.com", "password": "pw12345",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d (%s)", rec.Code, rec.Body.String())
	}
	var reg authapi.AuthResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)
	token := reg.Tokens.AccessToken

	// Add show (imports from provider).
	rec = doJSON(t, e, http.MethodPost, "/me/shows", token, map[string]int64{"tmdb_id": 1399})
	if rec.Code != http.StatusCreated {
		t.Fatalf("add show: %d (%s)", rec.Code, rec.Body.String())
	}
	var us trackingapi.UserShow
	_ = json.Unmarshal(rec.Body.Bytes(), &us)
	showID := us.ShowId

	// Duplicate add → 409.
	rec = doJSON(t, e, http.MethodPost, "/me/shows", token, map[string]int64{"tmdb_id": 1399})
	if rec.Code != http.StatusConflict {
		t.Fatalf("dup add: want 409, got %d", rec.Code)
	}

	// List → 1 tracked, progress 0/2.
	rec = doJSON(t, e, http.MethodGet, "/me/shows", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d", rec.Code)
	}
	var list trackingapi.TrackedShowList
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Tracked) != 1 || list.Tracked[0].Show.SeasonCount == nil || *list.Tracked[0].Show.SeasonCount != 1 || list.Tracked[0].Progress.Total != 2 || list.Tracked[0].Progress.Watched != 0 {
		t.Fatalf("unexpected list: %+v", list.Tracked)
	}

	// Fetch episode ids from the catalog detail.
	rec = doJSON(t, e, http.MethodGet, "/shows/"+strconv.FormatInt(showID, 10), token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get show: %d", rec.Code)
	}
	var show showsapi.Show
	_ = json.Unmarshal(rec.Body.Bytes(), &show)
	if show.Seasons == nil || len((*show.Seasons)[0].Episodes) != 2 {
		t.Fatalf("unexpected show detail: %+v", show)
	}
	ep1 := (*show.Seasons)[0].Episodes[0].Id
	ep2 := (*show.Seasons)[0].Episodes[1].Id

	// Watch episode 1.
	rec = doJSON(t, e, http.MethodPost,
		"/me/shows/"+strconv.FormatInt(showID, 10)+"/episodes/"+strconv.FormatInt(ep1, 10)+"/watch", token, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("watch: %d (%s)", rec.Code, rec.Body.String())
	}

	// Progress now 1/2, next = ep2.
	rec = doJSON(t, e, http.MethodGet, "/me/shows/"+strconv.FormatInt(showID, 10), token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get tracked: %d", rec.Code)
	}
	var ts trackingapi.TrackedShow
	_ = json.Unmarshal(rec.Body.Bytes(), &ts)
	if ts.Progress.Watched != 1 || ts.Progress.NextUnwatchedEpisodeId == nil || *ts.Progress.NextUnwatchedEpisodeId != ep2 {
		t.Fatalf("unexpected progress: %+v", ts.Progress)
	}

	// Whole-show marking is idempotent and requires ownership/authentication.
	watchPath := "/me/shows/" + strconv.FormatInt(showID, 10) + "/watch"
	if rec := doJSON(t, e, http.MethodPost, watchPath, "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated watch: %d", rec.Code)
	}
	if rec := doJSON(t, e, http.MethodPost, "/me/shows/999999/watch", token, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("missing tracked show: %d", rec.Code)
	}
	for i := 0; i < 2; i++ {
		if rec := doJSON(t, e, http.MethodPost, watchPath, token, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("watch show: %d (%s)", rec.Code, rec.Body.String())
		}
	}
	rec = doJSON(t, e, http.MethodGet, "/me/shows/"+strconv.FormatInt(showID, 10), token, nil)
	var completedShow trackingapi.TrackedShow
	if err := json.Unmarshal(rec.Body.Bytes(), &completedShow); err != nil {
		t.Fatal(err)
	}
	// WatchShow marks every aired episode but leaves the user's status alone
	// (it stays "watching" from AddShow, not "completed").
	if completedShow.Progress.Watched != 2 || len(completedShow.Progress.WatchedEpisodeIds) != 2 || completedShow.Progress.NextUnwatchedEpisodeId != nil || completedShow.UserShow.Status != "watching" {
		t.Fatalf("whole show progress: %+v", completedShow)
	}

	// A reference value can be edited without recreating the link.
	linksPath := "/me/shows/" + strconv.FormatInt(showID, 10) + "/links"
	rec = doJSON(t, e, http.MethodPost, linksPath, token, map[string]string{"kind": "wiki", "url": "https://example.com", "label": "Reference"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("add link: %d (%s)", rec.Code, rec.Body.String())
	}
	var link trackingapi.Link
	if err := json.Unmarshal(rec.Body.Bytes(), &link); err != nil {
		t.Fatal(err)
	}
	linkPath := linksPath + "/" + strconv.FormatInt(link.Id, 10)
	rec = doJSON(t, e, http.MethodPatch, linkPath, "", map[string]string{"url": "text"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated edit: %d", rec.Code)
	}
	rec = doJSON(t, e, http.MethodPatch, linkPath, token, map[string]string{"url": "На диске"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("edit link: %d (%s)", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, e, http.MethodGet, linksPath, token, nil)
	var links trackingapi.LinkList
	if err := json.Unmarshal(rec.Body.Bytes(), &links); err != nil {
		t.Fatal(err)
	}
	if len(links.Links) != 1 || links.Links[0].Id != link.Id || links.Links[0].Url != "На диске" {
		t.Fatalf("links after edit: %+v", links)
	}
	rec = doJSON(t, e, http.MethodPatch, linksPath+"/999999", token, map[string]string{"url": "text"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing link: %d", rec.Code)
	}

	// Update status → completed.
	rec = doJSON(t, e, http.MethodPatch, "/me/shows/"+strconv.FormatInt(showID, 10), token,
		map[string]string{"status": "completed"})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d", rec.Code)
	}

	// Rating: set to 8 → 200 with rating echoed.
	rec = doJSON(t, e, http.MethodPatch, "/me/shows/"+strconv.FormatInt(showID, 10), token,
		map[string]any{"rating": 8})
	if rec.Code != http.StatusOK {
		t.Fatalf("set rating: %d (%s)", rec.Code, rec.Body.String())
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &us)
	if us.Rating == nil || *us.Rating != 8 {
		t.Fatalf("rating not echoed: %+v", us.Rating)
	}
	// Invalid rating → 400.
	rec = doJSON(t, e, http.MethodPatch, "/me/shows/"+strconv.FormatInt(showID, 10), token,
		map[string]any{"rating": 11})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid rating: want 400, got %d", rec.Code)
	}
	// Rating must NOT leak through the public /users/{id}/shows surface.
	rec = doJSON(t, e, http.MethodGet, "/users/"+strconv.FormatInt(reg.User.Id, 10)+"/shows", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("user shows: %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "rating") {
		t.Fatalf("/users/{id}/shows leaked rating: %s", rec.Body.String())
	}
	// Clear rating → 200, back to null (fresh struct: cleared rating is omitted from JSON).
	rec = doJSON(t, e, http.MethodPatch, "/me/shows/"+strconv.FormatInt(showID, 10), token,
		map[string]any{"clear_rating": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("clear rating: %d", rec.Code)
	}
	var afterClear trackingapi.UserShow
	_ = json.Unmarshal(rec.Body.Bytes(), &afterClear)
	if afterClear.Rating != nil {
		t.Fatalf("rating not cleared: %+v", afterClear.Rating)
	}

	// Settings roundtrip, including an editable display name.
	rec = doJSON(t, e, http.MethodPatch, "/me/settings", token, map[string]string{"timezone": "Europe/Moscow", "display_name": "  Slice User  "})
	if rec.Code != http.StatusOK {
		t.Fatalf("update settings: %d (%s)", rec.Code, rec.Body.String())
	}
	var updated trackingapi.Settings
	_ = json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated.DisplayName != "Slice User" {
		t.Fatalf("display name not trimmed/echoed: %+v", updated)
	}
	rec = doJSON(t, e, http.MethodGet, "/me/settings", token, nil)
	var settings trackingapi.Settings
	_ = json.Unmarshal(rec.Body.Bytes(), &settings)
	if settings.Timezone != "Europe/Moscow" || settings.DisplayName != "Slice User" {
		t.Fatalf("settings: %+v", settings)
	}

	// Blank display name resets to the derived default (email local part).
	rec = doJSON(t, e, http.MethodPatch, "/me/settings", token, map[string]string{"timezone": "Europe/Moscow", "display_name": "   "})
	if rec.Code != http.StatusOK {
		t.Fatalf("reset display name: %d (%s)", rec.Code, rec.Body.String())
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated.DisplayName != "slice" {
		t.Fatalf("blank display name should fall back to email local part, got %+v", updated)
	}

	// Over-long display name → 400.
	rec = doJSON(t, e, http.MethodPatch, "/me/settings", token, map[string]string{"timezone": "Europe/Moscow", "display_name": strings.Repeat("я", 51)})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("over-long display name: want 400, got %d", rec.Code)
	}

	// Unauthenticated access → 401.
	rec = doJSON(t, e, http.MethodGet, "/me/shows", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: want 401, got %d", rec.Code)
	}

	// Remove.
	rec = doJSON(t, e, http.MethodDelete, "/me/shows/"+strconv.FormatInt(showID, 10), token, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("remove: %d", rec.Code)
	}
}
