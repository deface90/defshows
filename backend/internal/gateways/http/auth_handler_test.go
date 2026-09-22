package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	e, _ := newTestServerMail(t)
	return e
}

// captureMailer records the last email so tests can pull the reset link out.
type captureMailer struct {
	count int
	body  string
}

func (m *captureMailer) Send(_ context.Context, _, _, body string) error {
	m.count++
	m.body = body
	return nil
}

// newTestServerMail builds the auth router with a capturing mailer wired in.
func newTestServerMail(t *testing.T) (*echo.Echo, *captureMailer) {
	t.Helper()
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users")
	repo := repository.NewUserRepository(gdb)
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	mail := &captureMailer{}
	uc := usecase.NewAuthUsecase(repo, jwtMgr, time.Hour).WithMailer(mail, "https://app.example")
	return httpapi.NewAuthRouter(httpapi.NewAuthHandler(uc), jwtMgr), mail
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

	// Reusing the just-rotated token within the grace window is treated as a
	// benign race (concurrent tabs / reload) and reissued rather than logging
	// the user out. Genuine reuse past the grace window still revokes the family
	// — covered by the usecase unit test TestAuth_Refresh_RotationAndReuse.
	rec = doJSON(t, e, http.MethodPost, "/auth/refresh", "", map[string]string{
		"refresh_token": reg.Tokens.RefreshToken,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("reuse within grace: want 200, got %d", rec.Code)
	}
}

func TestAuthHandler_PasswordChange(t *testing.T) {
	e := newTestServer(t)

	rec := doJSON(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"email": "chg@example.com", "password": "oldpass1",
	})
	var reg authapi.AuthResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)

	// No token → 401.
	rec = doJSON(t, e, http.MethodPost, "/auth/password/change", "", map[string]string{
		"current_password": "oldpass1", "new_password": "newpass1",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("change no token: want 401, got %d", rec.Code)
	}

	// Wrong current password → 401.
	rec = doJSON(t, e, http.MethodPost, "/auth/password/change", reg.Tokens.AccessToken, map[string]string{
		"current_password": "wrong", "new_password": "newpass1",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("change wrong current: want 401, got %d", rec.Code)
	}

	// Success → 200 with a fresh token pair.
	rec = doJSON(t, e, http.MethodPost, "/auth/password/change", reg.Tokens.AccessToken, map[string]string{
		"current_password": "oldpass1", "new_password": "newpass1",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("change: want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	// New password logs in.
	rec = doJSON(t, e, http.MethodPost, "/auth/login", "", map[string]string{
		"email": "chg@example.com", "password": "newpass1",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login new pw: want 200, got %d", rec.Code)
	}
}

func TestAuthHandler_PasswordReset(t *testing.T) {
	e, mail := newTestServerMail(t)

	rec := doJSON(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"email": "rst@example.com", "password": "oldpass1",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d", rec.Code)
	}

	// Forgot for an unknown email still returns 204 (no enumeration) and sends nothing.
	rec = doJSON(t, e, http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": "nobody@example.com"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("forgot unknown: want 204, got %d", rec.Code)
	}
	if mail.count != 0 {
		t.Fatalf("unknown email must not send mail, got %d", mail.count)
	}

	// Forgot for the real email → 204 and one email with a reset link.
	rec = doJSON(t, e, http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": "rst@example.com"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("forgot: want 204, got %d", rec.Code)
	}
	if mail.count != 1 {
		t.Fatalf("want one email, got %d", mail.count)
	}
	i := strings.Index(mail.body, "token=")
	if i < 0 {
		t.Fatalf("no token in email: %q", mail.body)
	}
	token := strings.FieldsFunc(mail.body[i+len("token="):], func(r rune) bool { return r == '\r' || r == '\n' || r == ' ' })[0]

	// Bad token → 400.
	rec = doJSON(t, e, http.MethodPost, "/auth/password/reset", "", map[string]string{
		"token": "bogus", "new_password": "newpass1",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reset bad token: want 400, got %d", rec.Code)
	}

	// Valid token → 204, and the new password logs in.
	rec = doJSON(t, e, http.MethodPost, "/auth/password/reset", "", map[string]string{
		"token": token, "new_password": "newpass1",
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("reset: want 204, got %d (%s)", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, e, http.MethodPost, "/auth/login", "", map[string]string{
		"email": "rst@example.com", "password": "newpass1",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login after reset: want 200, got %d", rec.Code)
	}
}
