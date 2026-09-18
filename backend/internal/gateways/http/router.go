package httpapi

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/deface90/defshows/backend/pkg/auth"
	adminapi "github.com/deface90/defshows/backend/pkg/server/admin"
	authapi "github.com/deface90/defshows/backend/pkg/server/auth"
	notesapi "github.com/deface90/defshows/backend/pkg/server/notes"
	notificationsapi "github.com/deface90/defshows/backend/pkg/server/notifications"
	showsapi "github.com/deface90/defshows/backend/pkg/server/shows"
	trackingapi "github.com/deface90/defshows/backend/pkg/server/tracking"
	usersapi "github.com/deface90/defshows/backend/pkg/server/users"
)

// authPublicPaths are the auth routes reachable without a valid access token.
var authPublicPaths = map[string]bool{
	"/auth/register":                 true,
	"/auth/login":                    true,
	"/auth/refresh":                  true,
	"/auth/logout":                   true,
	"/auth/oauth/:provider":          true,
	"/auth/oauth/:provider/callback": true,
	"/auth/oauth/exchange":           true,
}

// NewAuthRouter builds the echo router for the auth service. Every route except
// the public auth endpoints requires a valid Bearer access token.
func NewAuthRouter(h *AuthHandler, jwt *auth.JWTManager) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(requireAuthExcept(jwt, authPublicPaths))
	authapi.RegisterHandlers(e, h)
	return e
}

// NewWebRouter builds the echo router for the web service: auth, catalog, and
// tracking on one echo. Every route except the public auth endpoints requires a
// valid Bearer access token.
func NewWebRouter(authH *AuthHandler, showsH *ShowsHandler, trackingH *TrackingHandler, notificationsH *NotificationsHandler, notesH *NotesHandler, adminH *AdminHandler, usersH *UsersHandler, jwt *auth.JWTManager, corsOrigins ...string) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Recover())
	// CORS runs before the auth guard so browser preflight (OPTIONS) is answered
	// without a token.
	if len(corsOrigins) > 0 {
		e.Use(corsMiddleware(corsOrigins))
	}
	e.Use(requireAuthExcept(jwt, authPublicPaths))
	e.Use(requireAdminForAdminPaths())
	authapi.RegisterHandlers(e, authH)
	showsapi.RegisterHandlers(e, showsH)
	trackingapi.RegisterHandlers(e, trackingH)
	notificationsapi.RegisterHandlers(e, notificationsH)
	notesapi.RegisterHandlers(e, notesH)
	adminapi.RegisterHandlers(e, adminH)
	usersapi.RegisterHandlers(e, usersH)
	return e
}

// requireAdminForAdminPaths enforces the admin role on /admin/* routes (runs
// after auth, which populates claims).
func requireAdminForAdminPaths() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if strings.HasPrefix(c.Path(), "/admin") {
				claims, ok := auth.ClaimsFromContext(c)
				if !ok {
					return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
				}
				if claims.Role != string(entityRoleAdmin) {
					return echo.NewHTTPError(http.StatusForbidden, "forbidden")
				}
			}
			return next(c)
		}
	}
}

const entityRoleAdmin = "admin"

// corsMiddleware allows the given browser origins to call the API. Auth uses a
// Bearer header (not cookies), so credentials are not enabled.
func corsMiddleware(origins []string) echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: origins,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType},
	})
}

// requireAuthExcept enforces JWT auth on all routes whose matched path is not in
// the public set.
func requireAuthExcept(jwt *auth.JWTManager, public map[string]bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if public[c.Path()] {
				return next(c)
			}
			return jwt.RequireAuth(next)(c)
		}
	}
}
