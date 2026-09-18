package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	httpapi "github.com/deface90/defshows/backend/internal/gateways/http"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/internal/testutil"
	"github.com/deface90/defshows/backend/pkg/auth"
	authapi "github.com/deface90/defshows/backend/pkg/server/auth"
)

func newTestServer(t *testing.T) *echo.Echo {
	t.Helper()
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users")
	repo := repository.NewUserRepository(gdb)
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	uc := usecase.NewAuthUsecase(repo, jwtMgr, time.Hour)
	return httpapi.NewAuthRouter(httpapi.NewAuthHandler(uc), jwtMgr)
}

func doJSON(t *testing.T, e *echo.Echo, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAuthHandler_Flow(t *testing.T) {
	e := newTestServer(t)

	// Register.
	rec := doJSON(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"email": "flow@example.com", "password": "pw12345",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var reg authapi.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &reg); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	if reg.Tokens.AccessToken == "" || reg.User.Id == 0 {
		t.Fatalf("unexpected register payload: %+v", reg)
	}

	// Duplicate → 409.
	rec = doJSON(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"email": "flow@example.com", "password": "pw12345",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("dup register: want 409, got %d", rec.Code)
	}

	// Wrong password → 401.
	rec = doJSON(t, e, http.MethodPost, "/auth/login", "", map[string]string{
		"email": "flow@example.com", "password": "nope",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad login: want 401, got %d", rec.Code)
	}

	// /auth/me without token → 401.
	rec = doJSON(t, e, http.MethodGet, "/auth/me", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me no token: want 401, got %d", rec.Code)
	}

	// /auth/me with token → 200.
	rec = doJSON(t, e, http.MethodGet, "/auth/me", reg.Tokens.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("me: want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var me authapi.User
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	if me.Email == nil || *me.Email != "flow@example.com" {
		t.Fatalf("unexpected me: %+v", me)
	}
}

func TestAuthHandler_RefreshRotation(t *testing.T) {
	e := newTestServer(t)

	rec := doJSON(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"email": "rot@example.com", "password": "pw12345",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d", rec.Code)
	}
	var reg authapi.AuthResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)

	// Rotate.
	rec = doJSON(t, e, http.MethodPost, "/auth/refresh", "", map[string]string{
		"refresh_token": reg.Tokens.RefreshToken,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh: want 200, got %d", rec.Code)
	}

	// Reuse old refresh → 401.
	rec = doJSON(t, e, http.MethodPost, "/auth/refresh", "", map[string]string{
		"refresh_token": reg.Tokens.RefreshToken,
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("reuse refresh: want 401, got %d", rec.Code)
	}
}
