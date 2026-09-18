package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCORSMiddleware(t *testing.T) {
	e := echo.New()
	e.Use(corsMiddleware([]string{"http://localhost:3000"}))
	e.GET("/ping", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	// Allowed origin → echoed back in the header.
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(echo.HeaderOrigin, "http://localhost:3000")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if got := rec.Header().Get(echo.HeaderAccessControlAllowOrigin); got != "http://localhost:3000" {
		t.Fatalf("allowed origin: want header set, got %q", got)
	}

	// Preflight OPTIONS is answered without hitting the route/auth.
	pre := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	pre.Header.Set(echo.HeaderOrigin, "http://localhost:3000")
	pre.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
	preRec := httptest.NewRecorder()
	e.ServeHTTP(preRec, pre)
	if preRec.Code != http.StatusNoContent {
		t.Fatalf("preflight: want 204, got %d", preRec.Code)
	}

	// Disallowed origin → no allow-origin header.
	bad := httptest.NewRequest(http.MethodGet, "/ping", nil)
	bad.Header.Set(echo.HeaderOrigin, "http://evil.example")
	badRec := httptest.NewRecorder()
	e.ServeHTTP(badRec, bad)
	if got := badRec.Header().Get(echo.HeaderAccessControlAllowOrigin); got != "" {
		t.Fatalf("disallowed origin: want empty header, got %q", got)
	}
}
