package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/deface90/defshows/backend/pkg/auth"
)

func okHandler(c echo.Context) error { return c.String(http.StatusOK, "ok") }

func TestRequireAuth(t *testing.T) {
	m := auth.NewJWTManager("secret", time.Hour)
	token, _ := m.Generate(7, "user")

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{name: "no header", authHeader: "", wantStatus: http.StatusUnauthorized},
		{name: "not bearer", authHeader: "Basic abc", wantStatus: http.StatusUnauthorized},
		{name: "bad token", authHeader: "Bearer nope", wantStatus: http.StatusUnauthorized},
		{name: "valid", authHeader: "Bearer " + token, wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := m.RequireAuth(okHandler)
			if err := h(c); err != nil {
				e.HTTPErrorHandler(err, c)
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("want %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestOptionalAuth(t *testing.T) {
	m := auth.NewJWTManager("secret", time.Hour)
	token, _ := m.Generate(7, "user")

	tests := []struct {
		name       string
		authHeader string
		wantClaims bool
	}{
		{name: "no header passes as guest", authHeader: "", wantClaims: false},
		{name: "invalid token passes as guest", authHeader: "Bearer nope", wantClaims: false},
		{name: "valid token populates claims", authHeader: "Bearer " + token, wantClaims: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			var gotClaims bool
			h := m.OptionalAuth(func(c echo.Context) error {
				_, gotClaims = auth.ClaimsFromContext(c)
				return c.String(http.StatusOK, "ok")
			})
			if err := h(c); err != nil {
				e.HTTPErrorHandler(err, c)
			}
			// OptionalAuth never rejects: always reaches the handler with 200.
			if rec.Code != http.StatusOK {
				t.Fatalf("want 200, got %d", rec.Code)
			}
			if gotClaims != tt.wantClaims {
				t.Fatalf("claims present: want %v, got %v", tt.wantClaims, gotClaims)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	m := auth.NewJWTManager("secret", time.Hour)

	tests := []struct {
		name       string
		role       string
		wantStatus int
	}{
		{name: "admin allowed", role: "admin", wantStatus: http.StatusOK},
		{name: "user forbidden", role: "user", wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _ := m.Generate(1, tt.role)
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// chain: RequireAuth -> RequireRole("admin") -> okHandler
			h := m.RequireAuth(auth.RequireRole("admin")(okHandler))
			if err := h(c); err != nil {
				e.HTTPErrorHandler(err, c)
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("want %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}
