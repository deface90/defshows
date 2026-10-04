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
	socialapi "github.com/deface90/defshows/backend/pkg/server/social"
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

func TestAdminHandler_Reports(t *testing.T) {
	e := newWebServer(t)

	// Two regular users: reporter files against target.
	reporterID, reporterTok := registerUser(t, e, "arep@example.com")
	targetID, _ := registerUser(t, e, "atgt@example.com")
	_ = reporterID

	rec := doJSON(t, e, http.MethodPost, "/me/reports", reporterTok, map[string]any{
		"target_user_id": targetID, "reason": "harassment", "note": "abusive dms",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create report: %d (%s)", rec.Code, rec.Body.String())
	}
	var created socialapi.Report
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	adminToken, _ := auth.NewJWTManager("test-secret", time.Hour).Generate(1, "admin")

	// Non-admin blocked by the router guard.
	if rec := doJSON(t, e, http.MethodGet, "/admin/reports", reporterTok, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin list: want 403, got %d", rec.Code)
	}

	// Admin list → 1 open report with enriched summaries.
	rec = doJSON(t, e, http.MethodGet, "/admin/reports?status=open", adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin list: %d (%s)", rec.Code, rec.Body.String())
	}
	var list adminapi.ReportList
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if list.Total != 1 || len(list.Reports) != 1 {
		t.Fatalf("want 1 report, got total=%d len=%d", list.Total, len(list.Reports))
	}
	if list.Reports[0].Target.Id != targetID || list.Reports[0].Reporter.Id != reporterID {
		t.Fatalf("summaries: %+v", list.Reports[0])
	}

	// Filtering by resolved yields nothing yet.
	rec = doJSON(t, e, http.MethodGet, "/admin/reports?status=resolved", adminToken, nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if list.Total != 0 {
		t.Fatalf("resolved filter: want 0, got %d", list.Total)
	}

	rid := strconv.FormatInt(created.Id, 10)

	// Invalid target statuses are rejected with 400 (only resolved/dismissed are
	// terminal; "open" and a bogus value are both outside the allowed set).
	for _, bad := range []string{"open", "bogus"} {
		if rec := doJSON(t, e, http.MethodPost, "/admin/reports/"+rid+"/resolve", adminToken,
			map[string]any{"status": bad}); rec.Code != http.StatusBadRequest {
			t.Fatalf("resolve status=%q: want 400, got %d (%s)", bad, rec.Code, rec.Body.String())
		}
	}

	// Resolve the report → 204.
	if rec := doJSON(t, e, http.MethodPost, "/admin/reports/"+rid+"/resolve", adminToken,
		map[string]any{"status": "resolved"}); rec.Code != http.StatusNoContent {
		t.Fatalf("resolve: want 204, got %d (%s)", rec.Code, rec.Body.String())
	}

	// It now appears under resolved.
	rec = doJSON(t, e, http.MethodGet, "/admin/reports?status=resolved", adminToken, nil)
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if list.Total != 1 || list.Reports[0].Status != "resolved" || list.Reports[0].ResolvedAt == nil {
		t.Fatalf("after resolve: %+v", list)
	}

	// Re-resolving an already-terminal report → 404 (no re-stamp).
	if rec := doJSON(t, e, http.MethodPost, "/admin/reports/"+rid+"/resolve", adminToken,
		map[string]any{"status": "dismissed"}); rec.Code != http.StatusNotFound {
		t.Fatalf("re-resolve terminal: want 404, got %d", rec.Code)
	}

	// Resolving a missing report → 404.
	if rec := doJSON(t, e, http.MethodPost, "/admin/reports/999999/resolve", adminToken,
		map[string]any{"status": "dismissed"}); rec.Code != http.StatusNotFound {
		t.Fatalf("resolve missing: want 404, got %d", rec.Code)
	}
}
