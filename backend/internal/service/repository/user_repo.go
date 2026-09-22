// Package repository provides gorm-backed data access.
package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/deface90/defshows/backend/internal/service/entity"
)

var (
	// ErrNotFound is returned when a row does not exist.
	ErrNotFound = errors.New("repository: not found")
	// ErrConflict is returned on a unique-constraint violation.
	ErrConflict = errors.New("repository: conflict")
)

// UserRepository is the data access layer for users, identities, and refresh
// tokens.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a UserRepository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserts a user. Returns ErrConflict if the email is taken.
func (r *UserRepository) CreateUser(ctx context.Context, u *entity.User) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrConflict
		}
		return err
	}
	return nil
}

// FindUserByEmail returns the user with the given email or ErrNotFound.
func (r *UserRepository) FindUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindUserByID returns the user with the given id or ErrNotFound.
func (r *UserRepository) FindUserByID(ctx context.Context, id int64) (*entity.User, error) {
	var u entity.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ListUsers searches displayed names and reads only the requested directory page.
func (r *UserRepository) ListUsers(ctx context.Context, query string, limit, offset int) ([]entity.User, int64, error) {
	const displayName = "COALESCE(NULLIF(display_name, ''), NULLIF(split_part(email, '@', 1), ''), 'Пользователь #' || id::text)"
	q := r.db.WithContext(ctx).Model(&entity.User{})
	if query != "" {
		literal := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query)
		q = q.Where(displayName+" ILIKE ?", "%"+literal+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []entity.User
	err := q.Order("LOWER(" + displayName + ") ASC, id ASC").Limit(limit).Offset(offset).Find(&users).Error
	return users, total, err
}

// SetPublic toggles a user's profile visibility.
func (r *UserRepository) SetPublic(ctx context.Context, userID int64, public bool) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"is_public": public, "updated_at": time.Now()}).Error
}

// UpdateTimezone sets a user's timezone.
func (r *UserRepository) UpdateTimezone(ctx context.Context, userID int64, tz string) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"timezone": tz, "updated_at": time.Now()}).Error
}

// UpdatePasswordHash sets a user's password hash.
func (r *UserRepository) UpdatePasswordHash(ctx context.Context, userID int64, hash string) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"password_hash": hash, "updated_at": time.Now()}).Error
}

// RevokeUserTokens marks all still-active refresh tokens for a user revoked
// (used after a password change or reset to log every session out).
func (r *UserRepository) RevokeUserTokens(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Model(&entity.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

// CreatePasswordResetToken stores a single-use password-reset token.
func (r *UserRepository) CreatePasswordResetToken(ctx context.Context, userID int64, hash string, expiresAt time.Time) error {
	return r.db.WithContext(ctx).Create(&entity.PasswordResetToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: expiresAt,
	}).Error
}

// ConsumePasswordResetToken atomically validates and marks a reset token used,
// returning its user id. ErrNotFound if the token is missing, already used, or
// expired.
func (r *UserRepository) ConsumePasswordResetToken(ctx context.Context, hash string) (int64, error) {
	var userID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var t entity.PasswordResetToken
		if err := tx.Where("token_hash = ?", hash).First(&t).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if t.UsedAt != nil || time.Now().After(t.ExpiresAt) {
			return ErrNotFound
		}
		if err := tx.Model(&entity.PasswordResetToken{}).
			Where("id = ? AND used_at IS NULL", t.ID).
			Update("used_at", time.Now()).Error; err != nil {
			return err
		}
		userID = t.UserID
		return nil
	})
	return userID, err
}

// SetTelegramChatID links a Telegram chat to a user.
func (r *UserRepository) SetTelegramChatID(ctx context.Context, userID, chatID int64) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"telegram_chat_id": chatID, "updated_at": time.Now()}).Error
}

// TelegramChatID returns the user's linked Telegram chat id, or nil if unlinked.
func (r *UserRepository) TelegramChatID(ctx context.Context, userID int64) (*int64, error) {
	var u entity.User
	err := r.db.WithContext(ctx).Select("telegram_chat_id").
		Where("id = ?", userID).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u.TelegramChatID, nil
}

// FindIdentity returns a linked identity or ErrNotFound.
func (r *UserRepository) FindIdentity(ctx context.Context, provider, providerUserID string) (*entity.UserIdentity, error) {
	var id entity.UserIdentity
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_user_id = ?", provider, providerUserID).
		First(&id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// CreateIdentity links an external identity to a user.
func (r *UserRepository) CreateIdentity(ctx context.Context, id *entity.UserIdentity) error {
	if err := r.db.WithContext(ctx).Create(id).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrConflict
		}
		return err
	}
	return nil
}

// SaveRefreshToken persists a refresh token.
func (r *UserRepository) SaveRefreshToken(ctx context.Context, rt *entity.RefreshToken) error {
	return r.db.WithContext(ctx).Create(rt).Error
}

// FindRefreshByHash returns the refresh token with the given hash or ErrNotFound.
func (r *UserRepository) FindRefreshByHash(ctx context.Context, hash string) (*entity.RefreshToken, error) {
	var rt entity.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&rt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

// RevokeRefreshToken marks a single token revoked.
func (r *UserRepository) RevokeRefreshToken(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&entity.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", time.Now()).Error
}

// RevokeFamily marks all still-active tokens in a family revoked (used on reuse
// detection and logout).
func (r *UserRepository) RevokeFamily(ctx context.Context, familyID string) error {
	return r.db.WithContext(ctx).
		Model(&entity.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", time.Now()).Error
}

// CreateOAuthHandoff stores a one-time OAuth handoff code.
func (r *UserRepository) CreateOAuthHandoff(ctx context.Context, code string, userID int64, expiresAt time.Time) error {
	return r.db.WithContext(ctx).Create(&entity.OAuthHandoff{Code: code, UserID: userID, ExpiresAt: expiresAt}).Error
}

// ConsumeOAuthHandoff atomically validates and deletes a handoff code.
func (r *UserRepository) ConsumeOAuthHandoff(ctx context.Context, code string) (int64, error) {
	var userID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var h entity.OAuthHandoff
		if err := tx.Where("code = ?", code).First(&h).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if err := tx.Where("code = ?", code).Delete(&entity.OAuthHandoff{}).Error; err != nil {
			return err
		}
		if time.Now().After(h.ExpiresAt) {
			return ErrNotFound
		}
		userID = h.UserID
		return nil
	})
	return userID, err
}
