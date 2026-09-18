// Package httpapi implements the echo HTTP handlers for defShows services.
package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/pkg/auth"
	"github.com/deface90/defshows/backend/pkg/oauth"
	authapi "github.com/deface90/defshows/backend/pkg/server/auth"
)

// AuthHandler implements the generated auth ServerInterface.
type AuthHandler struct {
	uc              *usecase.AuthUsecase
	oauth           oauth.Registry
	redirectBaseURL string
	frontendURL     string
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(uc *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{uc: uc, oauth: oauth.Registry{}}
}

// WithOAuth enables social login. redirectBaseURL is this service's public base
// URL (for building the provider callback), frontendURL is the SPA base.
func (h *AuthHandler) WithOAuth(registry oauth.Registry, redirectBaseURL, frontendURL string) *AuthHandler {
	h.oauth = registry
	h.redirectBaseURL = redirectBaseURL
	h.frontendURL = frontendURL
	return h
}

var _ authapi.ServerInterface = (*AuthHandler)(nil)

func toAPIUser(u *entity.User) authapi.User {
	return authapi.User{
		Id:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        authapi.UserRole(u.Role),
		Timezone:    u.Timezone,
	}
}

func toAPITokens(p *usecase.TokenPair) authapi.TokenPair {
	return authapi.TokenPair{AccessToken: p.AccessToken, RefreshToken: p.RefreshToken}
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(c echo.Context) error {
	var req authapi.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	u, pair, err := h.uc.Register(c.Request().Context(), string(req.Email), req.Password, c.Request().UserAgent())
	if err != nil {
		if errors.Is(err, usecase.ErrEmailTaken) {
			return echo.NewHTTPError(http.StatusConflict, "email already registered")
		}
		return err
	}
	return c.JSON(http.StatusCreated, authapi.AuthResponse{User: toAPIUser(u), Tokens: toAPITokens(pair)})
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c echo.Context) error {
	var req authapi.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	u, pair, err := h.uc.Login(c.Request().Context(), string(req.Email), req.Password, c.Request().UserAgent())
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCredentials) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}
		return err
	}
	return c.JSON(http.StatusOK, authapi.AuthResponse{User: toAPIUser(u), Tokens: toAPITokens(pair)})
}

// Refresh handles POST /auth/refresh.
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req authapi.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	pair, err := h.uc.Refresh(c.Request().Context(), req.RefreshToken, c.Request().UserAgent())
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidRefresh) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid refresh token")
		}
		return err
	}
	return c.JSON(http.StatusOK, toAPITokens(pair))
}

// Logout handles POST /auth/logout.
func (h *AuthHandler) Logout(c echo.Context) error {
	var req authapi.LogoutRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := h.uc.Logout(c.Request().Context(), req.RefreshToken); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// OauthRedirect handles GET /auth/oauth/{provider}.
func (h *AuthHandler) OauthRedirect(c echo.Context, provider string) error {
	p, ok := h.oauth.Get(provider)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "unknown provider")
	}
	state, err := randomState()
	if err != nil {
		return err
	}
	c.SetCookie(&http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/auth/oauth",
		HttpOnly: true,
		MaxAge:   600,
		SameSite: http.SameSiteLaxMode,
	})
	return c.Redirect(http.StatusFound, p.AuthURL(state, h.callbackURL(provider)))
}

// OauthCallback handles GET /auth/oauth/{provider}/callback.
func (h *AuthHandler) OauthCallback(c echo.Context, provider string, params authapi.OauthCallbackParams) error {
	p, ok := h.oauth.Get(provider)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "unknown provider")
	}
	// Validate state against the cookie.
	cookie, err := c.Cookie("oauth_state")
	if err != nil || params.State == nil || cookie.Value == "" || cookie.Value != *params.State {
		return h.redirectFrontendError(c, "invalid_state")
	}
	if params.Code == nil || *params.Code == "" {
		return h.redirectFrontendError(c, "missing_code")
	}

	info, err := p.Exchange(c.Request().Context(), *params.Code, h.callbackURL(provider))
	if err != nil {
		return h.redirectFrontendError(c, "exchange_failed")
	}
	user, err := h.uc.LoginWithOAuth(c.Request().Context(), provider, info.ProviderUserID, info.Email, info.Name)
	if err != nil {
		return h.redirectFrontendError(c, "login_failed")
	}
	code, err := h.uc.CreateOAuthHandoff(c.Request().Context(), user.ID)
	if err != nil {
		return h.redirectFrontendError(c, "handoff_failed")
	}
	return c.Redirect(http.StatusFound, h.frontendURL+"/auth/callback?code="+url.QueryEscape(code))
}

// OauthExchange handles POST /auth/oauth/exchange.
func (h *AuthHandler) OauthExchange(c echo.Context) error {
	var req authapi.OAuthExchangeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	user, pair, err := h.uc.ExchangeOAuthHandoff(c.Request().Context(), req.Code, c.Request().UserAgent())
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidHandoff) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid handoff code")
		}
		return err
	}
	return c.JSON(http.StatusOK, authapi.AuthResponse{User: toAPIUser(user), Tokens: toAPITokens(pair)})
}

func (h *AuthHandler) callbackURL(provider string) string {
	return h.redirectBaseURL + "/auth/oauth/" + provider + "/callback"
}

func (h *AuthHandler) redirectFrontendError(c echo.Context, reason string) error {
	return c.Redirect(http.StatusFound, h.frontendURL+"/auth/callback?error="+url.QueryEscape(reason))
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GetMe handles GET /auth/me.
func (h *AuthHandler) GetMe(c echo.Context) error {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	u, err := h.uc.Me(c.Request().Context(), claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	return c.JSON(http.StatusOK, toAPIUser(u))
}
