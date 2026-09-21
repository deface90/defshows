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
	"github.com/deface90/defshows/backend/pkg/storage"
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
	// Catalog images are public (referenced directly by <img> tags).
	"/images/*":                    true,
	"/images/tmdb/:size/:filename": true,
}

// authOptionalPaths are catalog read routes reachable without a token. When a
// valid Bearer token is present its claims are parsed so handlers can enrich the
// response (e.g. "tracked" markers); an absent or invalid token passes through
// as a guest instead of returning 401.
var authOptionalPaths = map[string]bool{
	"/shows":                  true, // ListShows (local catalog search)
	"/shows/:id":              true, // GetShow
	"/shows/search":           true, // SearchShows
	"/shows/discover":         true, // DiscoverShows
	"/shows/discover/filters": true, // GetDiscoveryFilters
	"/shows/tmdb/:tmdb_id":    true, // GetShowByTMDB
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
// tracking on one echo. Public auth endpoints and catalog images do not require
// a Bearer access token.
func NewWebRouter(authH *AuthHandler, showsH *ShowsHandler, trackingH *TrackingHandler, notificationsH *NotificationsHandler, notesH *NotesHandler, adminH *AdminHandler, usersH *UsersHandler, jwt *auth.JWTManager, imageStore *storage.S3, imageClient *http.Client, corsOrigins ...string) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Recover())
	// CORS runs before the auth guard so browser preflight (OPTIONS) is answered
	// without a token.
	if len(corsOrigins) > 0 {
		e.Use(corsMiddleware(corsOrigins))
	}
	e.Use(authGuard(jwt, authPublicPaths, authOptionalPaths))
	e.Use(requireAdminForAdminPaths())
	e.GET("/images/tmdb/:size/:filename", serveTMDBImage(imageClient))
	if imageStore != nil {
		e.GET("/images/*", serveImage(imageStore))
	}
	authapi.RegisterHandlers(e, authH)
	showsapi.RegisterHandlers(e, showsH)
	trackingapi.RegisterHandlers(e, trackingH)
	notificationsapi.RegisterHandlers(e, notificationsH)
	notesapi.RegisterHandlers(e, notesH)
	adminapi.RegisterHandlers(e, adminH)
	usersapi.RegisterHandlers(e, usersH)
	return e
}

// serveImage streams a mirrored object from storage. The key is the path after
// "/images/" (e.g. shows/42/poster.jpg).
func serveImage(store *storage.S3) echo.HandlerFunc {
	return func(c echo.Context) error {
		key := c.Param("*")
		if key == "" {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}
		body, contentType, err := store.Get(c.Request().Context(), key)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}
		defer body.Close()
		c.Response().Header().Set("Cache-Control", "public, max-age=86400")
		return c.Stream(http.StatusOK, contentType, body)
	}
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

// authGuard classifies each route into one of three tiers by its matched path:
// fully public (no token needed), optional-auth (token parsed if present, never
// rejected), or protected (valid token required). It replaces the binary
// public/protected split for routers that expose public catalog reads.
func authGuard(jwt *auth.JWTManager, public, optional map[string]bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			switch {
			case public[c.Path()]:
				return next(c)
			case optional[c.Path()]:
				return jwt.OptionalAuth(next)(c)
			default:
				return jwt.RequireAuth(next)(c)
			}
		}
	}
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
