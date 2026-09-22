package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/deface90/defshows/backend/internal/service/entity"
)

// NotificationRepository is the data access layer for notifications.
type NotificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository creates a NotificationRepository.
func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// GetPrefs returns a user's prefs, or defaults (not persisted) if none exist.
func (r *NotificationRepository) GetPrefs(ctx context.Context, userID int64) (*entity.NotificationPref, error) {
	var p entity.NotificationPref
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		def := entity.DefaultNotificationPref(userID)
		return &def, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpsertPrefs inserts or updates a user's prefs.
func (r *NotificationRepository) UpsertPrefs(ctx context.Context, p *entity.NotificationPref) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"episode_release", "season_start", "weekly_digest", "channel", "lead_time_hours"}),
	}).Create(p).Error
}

// CreateNotificationIfAbsent inserts a notification, ignoring duplicates by
// dedupe_key. Returns true if a row was created.
func (r *NotificationRepository) CreateNotificationIfAbsent(ctx context.Context, n *entity.Notification) (bool, error) {
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "dedupe_key"}},
		DoNothing: true,
	}).Create(n)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ListForUser returns a user's notification feed, newest first.
func (r *NotificationRepository) ListForUser(ctx context.Context, userID int64, limit int) ([]entity.Notification, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var out []entity.Notification
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Find(&out).Error
	return out, err
}

// MarkRead marks one of the user's notifications read (idempotent). Returns
// ErrNotFound if the notification doesn't exist or belongs to another user.
func (r *NotificationRepository) MarkRead(ctx context.Context, userID, id int64) error {
	res := r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("read_at", gorm.Expr("COALESCE(read_at, now())"))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkAllRead marks all of the user's unread notifications read.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Model(&entity.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", time.Now()).Error
}

// PendingNotification is a pending row joined with its delivery target.
// Target is the chat id (as text) for telegram, or the device token for
// apns; ShowID is only populated for apns (used for the push's deep link).
type PendingNotification struct {
	ID     int64
	Target string
	Title  string
	Body   string
	ShowID *int64
}

// pendingQueries maps a channel name to the SQL that finds its pending,
// deliverable notifications.
var pendingQueries = map[string]string{
	"telegram": `
		SELECT n.id AS id, u.telegram_chat_id::text AS target, '' AS title, n.payload AS body, n.show_id AS show_id
		FROM notifications n
		JOIN users u ON u.id = n.user_id
		WHERE n.status = 'pending'
		  AND n.channel = 'telegram'
		  AND n.scheduled_for <= now()
		  AND u.telegram_chat_id IS NOT NULL
		ORDER BY n.scheduled_for
		LIMIT ?`,
	"apns": `
		SELECT n.id AS id, u.apns_device_token AS target, 'defShows' AS title, n.payload AS body, n.show_id AS show_id
		FROM notifications n
		JOIN users u ON u.id = n.user_id
		WHERE n.status = 'pending'
		  AND n.channel = 'apns'
		  AND n.scheduled_for <= now()
		  AND u.apns_device_token IS NOT NULL
		ORDER BY n.scheduled_for
		LIMIT ?`,
}

// Pending returns pending notifications for a channel ("telegram" or "apns")
// ready to send.
func (r *NotificationRepository) Pending(ctx context.Context, channel string, limit int) ([]PendingNotification, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query, ok := pendingQueries[channel]
	if !ok {
		return nil, fmt.Errorf("repository: unknown notification channel %q", channel)
	}
	var out []PendingNotification
	err := r.db.WithContext(ctx).Raw(query, limit).Scan(&out).Error
	return out, err
}

// MarkSent marks a notification sent.
func (r *NotificationRepository) MarkSent(ctx context.Context, id int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&entity.Notification{}).Where("id = ?", id).
		Updates(map[string]any{"status": entity.NotifySent, "sent_at": now}).Error
}

// MarkFailed marks a notification failed.
func (r *NotificationRepository) MarkFailed(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&entity.Notification{}).Where("id = ?", id).
		Update("status", entity.NotifyFailed).Error
}

// EventCandidate is a user+episode pair that should be notified about, along
// with the channels currently available to reach that user.
type EventCandidate struct {
	UserID         int64
	ShowID         int64
	EpisodeID      int64
	ShowTitle      string
	SeasonNumber   int
	EpisodeNumber  int
	TelegramChatID *int64
	APNsToken      *string
}

// ReleasedEpisodeCandidates finds aired-but-unwatched episodes for watching
// users who have episode_release enabled and at least one delivery channel
// linked.
func (r *NotificationRepository) ReleasedEpisodeCandidates(ctx context.Context, since time.Time) ([]EventCandidate, error) {
	var out []EventCandidate
	err := r.db.WithContext(ctx).Raw(`
		SELECT us.user_id           AS user_id,
		       e.show_id            AS show_id,
		       e.id                 AS episode_id,
		       s.title              AS show_title,
		       e.season_number      AS season_number,
		       e.episode_number     AS episode_number,
		       u.telegram_chat_id   AS telegram_chat_id,
		       u.apns_device_token  AS apns_token
		FROM episodes e
		JOIN shows s       ON s.id = e.show_id
		JOIN user_shows us ON us.show_id = e.show_id AND us.status = 'watching'
		JOIN users u       ON u.id = us.user_id
		LEFT JOIN notification_prefs p ON p.user_id = us.user_id
		LEFT JOIN user_episodes ue ON ue.user_show_id = us.id AND ue.episode_id = e.id AND ue.watched = true
		WHERE e.air_date IS NOT NULL
		  AND e.air_date <= now()::date
		  AND e.air_date >= ?
		  AND COALESCE(p.episode_release, true) = true
		  AND (u.telegram_chat_id IS NOT NULL OR u.apns_device_token IS NOT NULL)
		  AND ue.id IS NULL`, since).Scan(&out).Error
	return out, err
}

// EpisodeUpcomingCandidates finds not-yet-aired episodes of watched shows
// whose air date falls within each user's configured lead time.
func (r *NotificationRepository) EpisodeUpcomingCandidates(ctx context.Context, now time.Time) ([]EventCandidate, error) {
	var out []EventCandidate
	err := r.db.WithContext(ctx).Raw(`
		SELECT us.user_id           AS user_id,
		       e.show_id            AS show_id,
		       e.id                 AS episode_id,
		       s.title              AS show_title,
		       e.season_number      AS season_number,
		       e.episode_number     AS episode_number,
		       u.telegram_chat_id   AS telegram_chat_id,
		       u.apns_device_token  AS apns_token
		FROM episodes e
		JOIN shows s       ON s.id = e.show_id
		JOIN user_shows us ON us.show_id = e.show_id AND us.status = 'watching'
		JOIN users u       ON u.id = us.user_id
		LEFT JOIN notification_prefs p ON p.user_id = us.user_id
		WHERE e.air_date IS NOT NULL
		  AND e.air_date::timestamptz > ?::timestamptz
		  AND e.air_date::timestamptz <= (?::timestamptz + (COALESCE(p.lead_time_hours, 24) || ' hours')::interval)
		  AND COALESCE(p.episode_release, true) = true
		  AND (u.telegram_chat_id IS NOT NULL OR u.apns_device_token IS NOT NULL)`, now, now).Scan(&out).Error
	return out, err
}

// SeasonUpcomingCandidates finds not-yet-aired season premieres (episode 1)
// of watched shows whose air date falls within each user's configured lead
// time.
func (r *NotificationRepository) SeasonUpcomingCandidates(ctx context.Context, now time.Time) ([]EventCandidate, error) {
	var out []EventCandidate
	err := r.db.WithContext(ctx).Raw(`
		SELECT us.user_id           AS user_id,
		       e.show_id            AS show_id,
		       e.id                 AS episode_id,
		       s.title              AS show_title,
		       e.season_number      AS season_number,
		       e.episode_number     AS episode_number,
		       u.telegram_chat_id   AS telegram_chat_id,
		       u.apns_device_token  AS apns_token
		FROM episodes e
		JOIN shows s       ON s.id = e.show_id
		JOIN user_shows us ON us.show_id = e.show_id AND us.status = 'watching'
		JOIN users u       ON u.id = us.user_id
		LEFT JOIN notification_prefs p ON p.user_id = us.user_id
		WHERE e.air_date IS NOT NULL
		  AND e.episode_number = 1
		  AND e.air_date::timestamptz > ?::timestamptz
		  AND e.air_date::timestamptz <= (?::timestamptz + (COALESCE(p.lead_time_hours, 24) || ' hours')::interval)
		  AND COALESCE(p.season_start, true) = true
		  AND (u.telegram_chat_id IS NOT NULL OR u.apns_device_token IS NOT NULL)`, now, now).Scan(&out).Error
	return out, err
}

// CreateLinkToken stores a one-time Telegram link token.
func (r *NotificationRepository) CreateLinkToken(ctx context.Context, t *entity.TelegramLinkToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// ConsumeLinkToken atomically validates and deletes a token, returning its user.
func (r *NotificationRepository) ConsumeLinkToken(ctx context.Context, token string) (int64, error) {
	var userID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var t entity.TelegramLinkToken
		if err := tx.Where("token = ?", token).First(&t).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if err := tx.Where("token = ?", token).Delete(&entity.TelegramLinkToken{}).Error; err != nil {
			return err
		}
		if time.Now().After(t.ExpiresAt) {
			return ErrNotFound
		}
		userID = t.UserID
		return nil
	})
	return userID, err
}
