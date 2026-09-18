package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const contextKeyClaims = "auth_claims"

// RequireAuth is echo middleware that validates the Bearer access token and
// stores the resulting claims in the request context.
func (m *JWTManager) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		header := c.Request().Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
		}
		claims, err := m.Parse(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		c.Set(contextKeyClaims, claims)
		return next(c)
	}
}

// RequireRole is echo middleware that enforces a role. It must run after
// RequireAuth.
func RequireRole(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := ClaimsFromContext(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
			}
			if claims.Role != role {
				return echo.NewHTTPError(http.StatusForbidden, "forbidden")
			}
			return next(c)
		}
	}
}

// ClaimsFromContext returns the authenticated claims, if present.
func ClaimsFromContext(c echo.Context) (*Claims, bool) {
	claims, ok := c.Get(contextKeyClaims).(*Claims)
	return claims, ok
}
