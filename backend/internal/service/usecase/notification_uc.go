package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
)

// ErrInvalidLinkToken is returned for an unknown/expired Telegram link token.
var ErrInvalidLinkToken = errors.New("usecase: invalid telegram link token")

// NotificationRepo is the storage dependency of NotificationUsecase.
type NotificationRepo interface {
	GetPrefs(ctx context.Context, userID int64) (*entity.NotificationPref, error)
	UpsertPrefs(ctx context.Context, p *entity.NotificationPref) error
	ListForUser(ctx context.Context, userID int64, limit int) ([]entity.Notification, error)
	MarkRead(ctx context.Context, userID, id int64) error
	MarkAllRead(ctx context.Context, userID int64) error
	CreateLinkToken(ctx context.Context, t *entity.TelegramLinkToken) error
	ConsumeLinkToken(ctx context.Context, token string) (int64, error)
}

// TelegramLinker reads and sets a user's Telegram chat id.
type TelegramLinker interface {
	SetTelegramChatID(ctx context.Context, userID, chatID int64) error
	TelegramChatID(ctx context.Context, userID int64) (*int64, error)
}

// DeviceTokenLinker reads and sets a user's APNs device token.
type DeviceTokenLinker interface {
	SetAPNsToken(ctx context.Context, userID int64, token string) error
	ClearAPNsToken(ctx context.Context, userID int64) error
}

// NotificationUsers is the user-storage dependency of NotificationUsecase.
type NotificationUsers interface {
	TelegramLinker
	DeviceTokenLinker
}

// NotificationUsecase implements the user-facing notification workflow.
type NotificationUsecase struct {
	repo        NotificationRepo
	users       NotificationUsers
	botUsername string
	linkTTL     time.Duration
}

// NewNotificationUsecase creates a NotificationUsecase.
func NewNotificationUsecase(repo NotificationRepo, users NotificationUsers, botUsername string, linkTTL time.Duration) *NotificationUsecase {
	if linkTTL <= 0 {
		linkTTL = 15 * time.Minute
	}
	return &NotificationUsecase{repo: repo, users: users, botUsername: botUsername, linkTTL: linkTTL}
}

// GetPrefs returns a user's notification preferences (defaults if unset).
func (uc *NotificationUsecase) GetPrefs(ctx context.Context, userID int64) (*entity.NotificationPref, error) {
	return uc.repo.GetPrefs(ctx, userID)
}

// UpdatePrefs updates a user's notification preferences.
func (uc *NotificationUsecase) UpdatePrefs(ctx context.Context, userID int64, episodeRelease, seasonStart, weeklyDigest *bool, leadTimeHours *int) (*entity.NotificationPref, error) {
	p, err := uc.repo.GetPrefs(ctx, userID)
	if err != nil {
		return nil, err
	}
	p.UserID = userID
	if episodeRelease != nil {
		p.EpisodeRelease = *episodeRelease
	}
	if seasonStart != nil {
		p.SeasonStart = *seasonStart
	}
	if weeklyDigest != nil {
		p.WeeklyDigest = *weeklyDigest
	}
	if leadTimeHours != nil {
		p.LeadTimeHours = *leadTimeHours
	}
	if p.Channel == "" {
		p.Channel = "telegram"
	}
	if err := uc.repo.UpsertPrefs(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Feed returns a user's notification feed.
func (uc *NotificationUsecase) Feed(ctx context.Context, userID int64, limit int) ([]entity.Notification, error) {
	return uc.repo.ListForUser(ctx, userID, limit)
}

// MarkRead marks a single notification read. Returns ErrNotFound if it doesn't
// belong to the user.
func (uc *NotificationUsecase) MarkRead(ctx context.Context, userID, id int64) error {
	return uc.repo.MarkRead(ctx, userID, id)
}

// MarkAllRead marks the whole feed read.
func (uc *NotificationUsecase) MarkAllRead(ctx context.Context, userID int64) error {
	return uc.repo.MarkAllRead(ctx, userID)
}

// IsTelegramLinked reports whether the user has a linked Telegram chat.
func (uc *NotificationUsecase) IsTelegramLinked(ctx context.Context, userID int64) (bool, error) {
	chatID, err := uc.users.TelegramChatID(ctx, userID)
	if err != nil {
		return false, err
	}
	return chatID != nil, nil
}

// GenerateTelegramLink issues a one-time deep link for the user to connect their
// Telegram account.
func (uc *NotificationUsecase) GenerateTelegramLink(ctx context.Context, userID int64) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	t := &entity.TelegramLinkToken{Token: token, UserID: userID, ExpiresAt: time.Now().Add(uc.linkTTL)}
	if err := uc.repo.CreateLinkToken(ctx, t); err != nil {
		return "", err
	}
	return fmt.Sprintf("https://t.me/%s?start=%s", uc.botUsername, token), nil
}

// RegisterDeviceToken links an APNs device token to the user, so they start
// receiving push notifications on that device.
func (uc *NotificationUsecase) RegisterDeviceToken(ctx context.Context, userID int64, token string) error {
	return uc.users.SetAPNsToken(ctx, userID, token)
}

// UnregisterDeviceToken unlinks the user's APNs device token (called on
// logout so a signed-out device stops receiving pushes for that account).
func (uc *NotificationUsecase) UnregisterDeviceToken(ctx context.Context, userID int64) error {
	return uc.users.ClearAPNsToken(ctx, userID)
}

// LinkTelegram consumes a link token and connects the chat to the user.
func (uc *NotificationUsecase) LinkTelegram(ctx context.Context, token string, chatID int64) error {
	userID, err := uc.repo.ConsumeLinkToken(ctx, token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidLinkToken
		}
		return err
	}
	return uc.users.SetTelegramChatID(ctx, userID, chatID)
}
