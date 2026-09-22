// Package usecase holds application business logic, decoupled from transport
// and (via interfaces) from storage.
package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/pkg/auth"
)

// Auth usecase errors.
var (
	ErrEmailTaken         = errors.New("usecase: email already registered")
	ErrInvalidCredentials = errors.New("usecase: invalid credentials")
	ErrInvalidRefresh     = errors.New("usecase: invalid refresh token")
	ErrInvalidHandoff     = errors.New("usecase: invalid oauth handoff code")
	ErrInvalidResetToken  = errors.New("usecase: invalid or expired reset token")
	ErrNoPassword         = errors.New("usecase: account has no password set")
)

// passwordResetTTL is how long an emailed reset link stays valid.
const passwordResetTTL = time.Hour

// Mailer sends transactional email (password-reset links). It is optional: when
// nil, RequestPasswordReset is a no-op beyond token creation.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// refreshReuseGrace tolerates benign refresh-token races: multiple tabs (or a
// reload firing while a refresh is in flight) can each present the same stored
// token near-simultaneously. Strict single-use rotation would treat the second
// presentation as reuse and revoke the whole family, logging the user out
// everywhere. Within this window after a token was rotated we instead issue a
// fresh pair in the same family. A genuinely stolen token replayed later (past
// the window) still trips reuse detection and kills the family.
const refreshReuseGrace = 30 * time.Second

// UserRepo is the storage dependency of AuthUsecase.
type UserRepo interface {
	CreateUser(ctx context.Context, u *entity.User) error
	FindUserByEmail(ctx context.Context, email string) (*entity.User, error)
	FindUserByID(ctx context.Context, id int64) (*entity.User, error)
	ListUsers(ctx context.Context, query string, limit, offset int) ([]entity.User, int64, error)
	UpdateTimezone(ctx context.Context, userID int64, tz string) error
	SetPublic(ctx context.Context, userID int64, public bool) error
	UpdatePasswordHash(ctx context.Context, userID int64, hash string) error
	SaveRefreshToken(ctx context.Context, rt *entity.RefreshToken) error
	FindRefreshByHash(ctx context.Context, hash string) (*entity.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id int64) error
	RevokeFamily(ctx context.Context, familyID string) error
	RevokeUserTokens(ctx context.Context, userID int64) error
	CreatePasswordResetToken(ctx context.Context, userID int64, hash string, expiresAt time.Time) error
	ConsumePasswordResetToken(ctx context.Context, hash string) (int64, error)
	FindIdentity(ctx context.Context, provider, providerUserID string) (*entity.UserIdentity, error)
	CreateIdentity(ctx context.Context, id *entity.UserIdentity) error
	CreateOAuthHandoff(ctx context.Context, code string, userID int64, expiresAt time.Time) error
	ConsumeOAuthHandoff(ctx context.Context, code string) (int64, error)
}

// TokenPair is an issued access + refresh token pair. RefreshToken is the raw
// (unhashed) value; only its hash is stored.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// AuthUsecase implements registration, login, and refresh-token rotation.
type AuthUsecase struct {
	repo        UserRepo
	jwt         *auth.JWTManager
	argon       auth.Argon2Params
	refreshTTL  time.Duration
	mailer      Mailer
	frontendURL string
}

// NewAuthUsecase creates an AuthUsecase.
func NewAuthUsecase(repo UserRepo, jwt *auth.JWTManager, refreshTTL time.Duration) *AuthUsecase {
	return &AuthUsecase{
		repo:       repo,
		jwt:        jwt,
		argon:      auth.DefaultArgon2Params(),
		refreshTTL: refreshTTL,
	}
}

// WithMailer enables emailed password reset. frontendURL is the SPA base used to
// build the reset link (${frontendURL}/reset-password?token=...).
func (uc *AuthUsecase) WithMailer(m Mailer, frontendURL string) *AuthUsecase {
	uc.mailer = m
	uc.frontendURL = frontendURL
	return uc
}

// Register creates a password account and returns an initial token pair.
func (uc *AuthUsecase) Register(ctx context.Context, email, password, userAgent string) (*entity.User, *TokenPair, error) {
	hash, err := auth.HashPassword(password, uc.argon)
	if err != nil {
		return nil, nil, err
	}
	u := &entity.User{
		Email:        &email,
		PasswordHash: &hash,
		Role:         entity.RoleUser,
		Timezone:     "UTC",
	}
	if err := uc.repo.CreateUser(ctx, u); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, nil, ErrEmailTaken
		}
		return nil, nil, err
	}
	pair, err := uc.issueTokens(ctx, u, userAgent, uuid.NewString(), nil)
	if err != nil {
		return nil, nil, err
	}
	return u, pair, nil
}

// SeedAdmin ensures an admin account with the given credentials exists. It is
// idempotent — an existing email is left untouched — and a blank email or
// password is a no-op. Returns true only when a new admin was created.
func (uc *AuthUsecase) SeedAdmin(ctx context.Context, email, password string) (bool, error) {
	if email == "" || password == "" {
		return false, nil
	}
	if _, err := uc.repo.FindUserByEmail(ctx, email); err == nil {
		return false, nil // already present
	} else if !errors.Is(err, repository.ErrNotFound) {
		return false, err
	}
	hash, err := auth.HashPassword(password, uc.argon)
	if err != nil {
		return false, err
	}
	u := &entity.User{
		Email:        &email,
		PasswordHash: &hash,
		Role:         entity.RoleAdmin,
		Timezone:     "UTC",
	}
	if err := uc.repo.CreateUser(ctx, u); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return false, nil // created concurrently by another instance
		}
		return false, err
	}
	return true, nil
}

// Login authenticates a password account and returns a token pair.
func (uc *AuthUsecase) Login(ctx context.Context, email, password, userAgent string) (*entity.User, *TokenPair, error) {
	u, err := uc.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if u.PasswordHash == nil {
		return nil, nil, ErrInvalidCredentials // social-only account
	}
	if err := auth.VerifyPassword(password, *u.PasswordHash); err != nil {
		return nil, nil, ErrInvalidCredentials
	}
	pair, err := uc.issueTokens(ctx, u, userAgent, uuid.NewString(), nil)
	if err != nil {
		return nil, nil, err
	}
	return u, pair, nil
}

// Refresh rotates a refresh token. It is single-use: the presented token is
// revoked and a new one issued in the same family. Presenting an already-revoked
// token is treated as reuse and revokes the whole family.
func (uc *AuthUsecase) Refresh(ctx context.Context, rawRefresh, userAgent string) (*TokenPair, error) {
	rt, err := uc.repo.FindRefreshByHash(ctx, hashToken(rawRefresh))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidRefresh
		}
		return nil, err
	}

	// Reuse detection: a revoked token presented again. If it was rotated only
	// moments ago this is a benign race (concurrent tabs / reload), so reissue
	// in the same family instead of revoking it. Presented later → genuine reuse,
	// kill the family.
	if rt.RevokedAt != nil {
		if time.Since(*rt.RevokedAt) <= refreshReuseGrace {
			u, err := uc.repo.FindUserByID(ctx, rt.UserID)
			if err != nil {
				return nil, err
			}
			return uc.issueTokens(ctx, u, userAgent, rt.FamilyID, &rt.ID)
		}
		_ = uc.repo.RevokeFamily(ctx, rt.FamilyID)
		return nil, ErrInvalidRefresh
	}
	if time.Now().After(rt.ExpiresAt) {
		return nil, ErrInvalidRefresh
	}

	u, err := uc.repo.FindUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return nil, err
	}
	return uc.issueTokens(ctx, u, userAgent, rt.FamilyID, &rt.ID)
}

// LoginWithOAuth finds or creates a user for a social identity. If the identity
// is new but the email matches an existing account, it links to that account.
func (uc *AuthUsecase) LoginWithOAuth(ctx context.Context, provider, providerUserID, email, name string) (*entity.User, error) {
	if id, err := uc.repo.FindIdentity(ctx, provider, providerUserID); err == nil {
		return uc.repo.FindUserByID(ctx, id.UserID)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	var user *entity.User
	if email != "" {
		u, err := uc.repo.FindUserByEmail(ctx, email)
		if err == nil {
			user = u
		} else if !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
	}
	if user == nil {
		var emailPtr *string
		if email != "" {
			emailPtr = &email
		}
		user = &entity.User{Email: emailPtr, DisplayName: name, Role: entity.RoleUser, Timezone: "UTC"}
		if err := uc.repo.CreateUser(ctx, user); err != nil {
			return nil, err
		}
	}
	if err := uc.repo.CreateIdentity(ctx, &entity.UserIdentity{
		UserID:         user.ID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	}); err != nil {
		return nil, err
	}
	return user, nil
}

// CreateOAuthHandoff issues a one-time code the SPA exchanges for tokens.
func (uc *AuthUsecase) CreateOAuthHandoff(ctx context.Context, userID int64) (string, error) {
	code, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := uc.repo.CreateOAuthHandoff(ctx, code, userID, time.Now().Add(5*time.Minute)); err != nil {
		return "", err
	}
	return code, nil
}

// ExchangeOAuthHandoff swaps a handoff code for a token pair.
func (uc *AuthUsecase) ExchangeOAuthHandoff(ctx context.Context, code, userAgent string) (*entity.User, *TokenPair, error) {
	userID, err := uc.repo.ConsumeOAuthHandoff(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrInvalidHandoff
		}
		return nil, nil, err
	}
	user, err := uc.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	pair, err := uc.issueTokens(ctx, user, userAgent, uuid.NewString(), nil)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}

// Me returns the user by id (for the /auth/me endpoint).
func (uc *AuthUsecase) Me(ctx context.Context, userID int64) (*entity.User, error) {
	return uc.repo.FindUserByID(ctx, userID)
}

// UpdateTimezone updates the user's timezone and returns the fresh user.
func (uc *AuthUsecase) UpdateTimezone(ctx context.Context, userID int64, tz string) (*entity.User, error) {
	if err := uc.repo.UpdateTimezone(ctx, userID, tz); err != nil {
		return nil, err
	}
	return uc.repo.FindUserByID(ctx, userID)
}

// SetProfileVisibility toggles the user's public/private profile and returns
// the fresh user.
func (uc *AuthUsecase) SetProfileVisibility(ctx context.Context, userID int64, public bool) (*entity.User, error) {
	if err := uc.repo.SetPublic(ctx, userID, public); err != nil {
		return nil, err
	}
	return uc.repo.FindUserByID(ctx, userID)
}

// ListUsers returns one filtered page of the public directory and its total size.
func (uc *AuthUsecase) ListUsers(ctx context.Context, query string, limit, offset int) ([]entity.User, int64, error) {
	return uc.repo.ListUsers(ctx, query, limit, offset)
}

// Logout revokes the family the presented refresh token belongs to.
func (uc *AuthUsecase) Logout(ctx context.Context, rawRefresh string) error {
	rt, err := uc.repo.FindRefreshByHash(ctx, hashToken(rawRefresh))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil // already gone; treat as success
		}
		return err
	}
	return uc.repo.RevokeFamily(ctx, rt.FamilyID)
}

// ChangePassword verifies the current password, sets a new one, revokes every
// existing session, and issues a fresh token pair so the caller stays logged in.
func (uc *AuthUsecase) ChangePassword(ctx context.Context, userID int64, current, newPassword, userAgent string) (*entity.User, *TokenPair, error) {
	u, err := uc.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if u.PasswordHash == nil {
		return nil, nil, ErrNoPassword // social-only account
	}
	if err := auth.VerifyPassword(current, *u.PasswordHash); err != nil {
		return nil, nil, ErrInvalidCredentials
	}
	hash, err := auth.HashPassword(newPassword, uc.argon)
	if err != nil {
		return nil, nil, err
	}
	if err := uc.repo.UpdatePasswordHash(ctx, userID, hash); err != nil {
		return nil, nil, err
	}
	if err := uc.repo.RevokeUserTokens(ctx, userID); err != nil {
		return nil, nil, err
	}
	pair, err := uc.issueTokens(ctx, u, userAgent, uuid.NewString(), nil)
	if err != nil {
		return nil, nil, err
	}
	return u, pair, nil
}

// RequestPasswordReset issues a reset token and emails a link. To avoid leaking
// which emails are registered it returns nil for unknown or social-only
// accounts (no token, no mail).
func (uc *AuthUsecase) RequestPasswordReset(ctx context.Context, email string) error {
	u, err := uc.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	if u.PasswordHash == nil || u.Email == nil {
		return nil // social-only: no password to reset
	}
	raw, err := randomToken()
	if err != nil {
		return err
	}
	if err := uc.repo.CreatePasswordResetToken(ctx, u.ID, hashToken(raw), time.Now().Add(passwordResetTTL)); err != nil {
		return err
	}
	if uc.mailer == nil {
		return nil
	}
	link := uc.frontendURL + "/reset-password?token=" + url.QueryEscape(raw)
	body := "Здравствуйте!\n\nВы запросили сброс пароля в defShows. " +
		"Перейдите по ссылке, чтобы задать новый пароль (действует 1 час):\n\n" +
		link + "\n\nЕсли вы этого не делали, просто проигнорируйте это письмо."
	return uc.mailer.Send(ctx, *u.Email, "Сброс пароля — defShows", body)
}

// ResetPassword consumes a reset token, sets the new password, and revokes every
// existing session.
func (uc *AuthUsecase) ResetPassword(ctx context.Context, token, newPassword string) error {
	userID, err := uc.repo.ConsumePasswordResetToken(ctx, hashToken(token))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidResetToken
		}
		return err
	}
	hash, err := auth.HashPassword(newPassword, uc.argon)
	if err != nil {
		return err
	}
	if err := uc.repo.UpdatePasswordHash(ctx, userID, hash); err != nil {
		return err
	}
	return uc.repo.RevokeUserTokens(ctx, userID)
}

func (uc *AuthUsecase) issueTokens(ctx context.Context, u *entity.User, userAgent, familyID string, prevTokenID *int64) (*TokenPair, error) {
	access, err := uc.jwt.Generate(u.ID, string(u.Role))
	if err != nil {
		return nil, err
	}
	raw, err := randomToken()
	if err != nil {
		return nil, err
	}
	rt := &entity.RefreshToken{
		UserID:      u.ID,
		TokenHash:   hashToken(raw),
		FamilyID:    familyID,
		PrevTokenID: prevTokenID,
		ExpiresAt:   time.Now().Add(uc.refreshTTL),
		UserAgent:   userAgent,
	}
	if err := uc.repo.SaveRefreshToken(ctx, rt); err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: raw}, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
