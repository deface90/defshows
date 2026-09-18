// Package entity holds the gorm domain models. Schema is owned by goose
// migrations; struct tags here are for mapping/reads, not AutoMigrate.
package entity

import "time"

// Role enumerates user roles.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// User is an account. Email/PasswordHash are nil for social-only accounts.
type User struct {
	ID             int64 `gorm:"primaryKey"`
	Email          *string
	PasswordHash   *string
	DisplayName    string
	Role           Role
	Timezone       string
	IsPublic       bool
	TelegramChatID *int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TableName maps User to the users table.
func (User) TableName() string { return "users" }

// UserIdentity links a user to an external OAuth identity.
type UserIdentity struct {
	ID             int64 `gorm:"primaryKey"`
	UserID         int64
	Provider       string
	ProviderUserID string
	CreatedAt      time.Time
}

// TableName maps UserIdentity to the user_identities table.
func (UserIdentity) TableName() string { return "user_identities" }

// RefreshToken is a single-use rotating refresh token. Tokens belonging to the
// same login form a family (FamilyID); PrevTokenID chains the rotation so reuse
// of a rotated token can be detected and the whole family revoked.
type RefreshToken struct {
	ID          int64 `gorm:"primaryKey"`
	UserID      int64
	TokenHash   string
	FamilyID    string
	PrevTokenID *int64
	ExpiresAt   time.Time
	RevokedAt   *time.Time
	UserAgent   string
	CreatedAt   time.Time
}

// TableName maps RefreshToken to the refresh_tokens table.
func (RefreshToken) TableName() string { return "refresh_tokens" }

// OAuthHandoff is a one-time code that lets the SPA fetch tokens after an OAuth
// login (so tokens never appear in a redirect URL).
type OAuthHandoff struct {
	Code      string `gorm:"primaryKey"`
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// TableName maps OAuthHandoff to the oauth_handoff_codes table.
func (OAuthHandoff) TableName() string { return "oauth_handoff_codes" }
