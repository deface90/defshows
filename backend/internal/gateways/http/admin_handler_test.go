package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/pkg/auth"
	adminapi "github.com/deface90/defshows/backend/pkg/server/admin"
	authapi "github.com/deface90/defshows/backend/pkg/server/auth"
)

func TestAdminHandler_RoleGuardAndCRUD(t *testing.T) {
	e := newWebServer(t)

	// Regular user is forbidden.
	rec := doJSON(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"email": "user@example.com", "password": "pw12345",
	})
	var reg authapi.AuthResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)
	rec = doJSON(t, e, http.MethodGet, "/admin/dubbing-studios", reg.Tokens.AccessToken, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("user on admin: want 403, got %d", rec.Code)
	}

	// Admin token (same secret as newWebServer).
	adminToken, _ := auth.NewJWTManager("test-secret", time.Hour).Generate(1, "admin")

	// Create.
	rec = doJSON(t, e, http.MethodPost, "/admin/dubbing-studios", adminToken,
		map[string]any{"name": "LostFilm", "site_url": "https://lostfilm.tv", "active": true})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d (%s)", rec.Code, rec.Body.String())
	}
	var ds adminapi.DubbingStudio
	_ = json.Unmarshal(rec.Body.Bytes(), &ds)

	// List.
	rec = doJSON(t, e, http.MethodGet, "/admin/dubbing-studios", adminToken, nil)
	var list adminapi.DubbingStudioList
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Studios) != 1 {
		t.Fatalf("want 1 studio, got %d", len(list.Studios))
	}

	// Update.
	rec = doJSON(t, e, http.MethodPut, "/admin/dubbing-studios/"+strconv.FormatInt(ds.Id, 10), adminToken,
		map[string]any{"name": "LostFilm HD", "active": false})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d", rec.Code)
	}

	// Delete + delete-again 404.
	rec = doJSON(t, e, http.MethodDelete, "/admin/dubbing-studios/"+strconv.FormatInt(ds.Id, 10), adminToken, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	rec = doJSON(t, e, http.MethodDelete, "/admin/dubbing-studios/"+strconv.FormatInt(ds.Id, 10), adminToken, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete again: want 404, got %d", rec.Code)
	}
}
