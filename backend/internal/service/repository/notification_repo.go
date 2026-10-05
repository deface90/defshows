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
		DoUpdates: clause.AssignmentColumns([]string{"episode_release", "season_start", "season_finale", "weekly_digest", "social_follows", "channel", "lead_time_hours"}),
	}).Create(p).Error
}

// EnqueueNotification inserts the notification (deduped by dedupe_key) and, when it is
// newly created, a pending delivery row per channel (in one transaction). With no channels
// the notification is feed-only. On a dedupe hit nothing is inserted (the deliveries were
// already enqueued on first insert). Returns true if the notification row was created.
func (r *NotificationRepository) EnqueueNotification(ctx context.Context, n *entity.Notification, channels []string) (bool, error) {
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "dedupe_key"}},
			DoNothing: true,
		}).Create(n)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // dedupe hit
		}
		created = true
		if len(channels) == 0 {
			return nil
		}
		now := time.Now()
		deliveries := make([]entity.NotificationDelivery, 0, len(channels))
		for _, ch := range channels {
			deliveries = append(deliveries, entity.NotificationDelivery{
				NotificationID: n.ID, Channel: ch, Status: entity.NotifyPending, ScheduledFor: now,
			})
		}
		return tx.Create(&deliveries).Error
	})
	return created, err
}

// DeleteByDedupeKeys removes notifications by their dedupe keys. Used to clear stale
// follow notifications on unfollow/reject so a later re-follow re-fires them.
func (r *NotificationRepository) DeleteByDedupeKeys(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("dedupe_key IN ?", keys).Delete(&entity.Notification{}).Error
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

// PendingNotification is a pending delivery joined with its event and target.
// ID is the notification_deliveries row id (marked sent/failed by the sender). Target is
// the chat id (as text) for telegram, or the device token for apns/fcm; ShowID feeds the
// push's deep link.
type PendingNotification struct {
	ID     int64
	Target string
	Title  string
	Body   string
	ShowID *int64
}

// pendingQueries maps a channel name to the SQL that finds its pending, deliverable rows.
// Each joins notification_deliveries → notifications (payload/show) → users (target).
var pendingQueries = map[string]string{
	"telegram": `
		SELECT d.id AS id, u.telegram_chat_id::text AS target, '' AS title, n.payload AS body, n.show_id AS show_id
		FROM notification_deliveries d
		JOIN notifications n ON n.id = d.notification_id
		JOIN users u ON u.id = n.user_id
		WHERE d.status = 'pending'
		  AND d.channel = 'telegram'
		  AND d.scheduled_for <= now()
		  AND u.telegram_chat_id IS NOT NULL
		ORDER BY d.scheduled_for
		LIMIT ?`,
	"apns": `
		SELECT d.id AS id, u.apns_device_token AS target, 'defShows' AS title, n.payload AS body, n.show_id AS show_id
		FROM notification_deliveries d
		JOIN notifications n ON n.id = d.notification_id
		JOIN users u ON u.id = n.user_id
		WHERE d.status = 'pending'
		  AND d.channel = 'apns'
		  AND d.scheduled_for <= now()
		  AND u.apns_device_token IS NOT NULL
		ORDER BY d.scheduled_for
		LIMIT ?`,
	"fcm": `
		SELECT d.id AS id, u.fcm_device_token AS target, 'defShows' AS title, n.payload AS body, n.show_id AS show_id
		FROM notification_deliveries d
		JOIN notifications n ON n.id = d.notification_id
		JOIN users u ON u.id = n.user_id
		WHERE d.status = 'pending'
		  AND d.channel = 'fcm'
		  AND d.scheduled_for <= now()
		  AND u.fcm_device_token IS NOT NULL
		ORDER BY d.scheduled_for
		LIMIT ?`,
}

// Pending returns pending notifications for a channel ("telegram", "apns" or
// "fcm") ready to send.
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

// MarkSent marks a delivery sent (id is a notification_deliveries row id).
func (r *NotificationRepository) MarkSent(ctx context.Context, id int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&entity.NotificationDelivery{}).Where("id = ?", id).
		Updates(map[string]any{"status": entity.NotifySent, "sent_at": now}).Error
}

// MarkFailed marks a delivery failed (id is a notification_deliveries row id).
func (r *NotificationRepository) MarkFailed(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&entity.NotificationDelivery{}).Where("id = ?", id).
		Update("status", entity.NotifyFailed).Error
}

// ClearTarget nulls a now-invalid delivery target for the channel, matched by value, so the
// sender stops using it (the device re-registers a fresh one on next launch). Matching by
// value (not user id) also covers a token that migrated between users.
func (r *NotificationRepository) ClearTarget(ctx context.Context, channel, target string) error {
	db := r.db.WithContext(ctx).Model(&entity.User{})
	switch channel {
	case "apns":
		return db.Where("apns_device_token = ?", target).Update("apns_device_token", nil).Error
	case "fcm":
		return db.Where("fcm_device_token = ?", target).Update("fcm_device_token", nil).Error
	case "telegram":
		return db.Where("telegram_chat_id::text = ?", target).Update("telegram_chat_id", nil).Error
	default:
		return nil
	}
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
	APNsToken      *string `gorm:"column:apns_token"`
	FCMToken       *string
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
		       u.apns_device_token  AS apns_token,
		       u.fcm_device_token   AS fcm_token
		FROM episodes e
		JOIN shows s       ON s.id = e.show_id
		JOIN user_shows us ON us.show_id = e.show_id AND us.status = 'watching'
		JOIN users u       ON u.id = us.user_id
		LEFT JOIN notification_prefs p ON p.user_id = us.user_id
		LEFT JOIN user_episodes ue ON ue.user_show_id = us.id AND ue.episode_id = e.id AND ue.watched = true
		WHERE e.air_date IS NOT NULL
		  AND e.air_date <= now()::date
		  AND e.air_date >= ?
		  AND COALESCE(p.episode_release, false) = true
		  AND (u.telegram_chat_id IS NOT NULL OR u.apns_device_token IS NOT NULL OR u.fcm_device_token IS NOT NULL)
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
		       u.apns_device_token  AS apns_token,
		       u.fcm_device_token   AS fcm_token
		FROM episodes e
		JOIN shows s       ON s.id = e.show_id
		JOIN user_shows us ON us.show_id = e.show_id AND us.status = 'watching'
		JOIN users u       ON u.id = us.user_id
		LEFT JOIN notification_prefs p ON p.user_id = us.user_id
		WHERE e.air_date IS NOT NULL
		  AND e.air_date::timestamptz > ?::timestamptz
		  AND e.air_date::timestamptz <= (?::timestamptz + (COALESCE(p.lead_time_hours, 24) || ' hours')::interval)
		  AND COALESCE(p.episode_release, false) = true
		  AND (u.telegram_chat_id IS NOT NULL OR u.apns_device_token IS NOT NULL OR u.fcm_device_token IS NOT NULL)`, now, now).Scan(&out).Error
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
		       u.apns_device_token  AS apns_token,
		       u.fcm_device_token   AS fcm_token
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
		  AND (u.telegram_chat_id IS NOT NULL OR u.apns_device_token IS NOT NULL OR u.fcm_device_token IS NOT NULL)`, now, now).Scan(&out).Error
	return out, err
}

// SeasonFinaleCandidates finds recently-aired season finales of watched shows
// for users who have season_finale enabled and at least one delivery channel.
// A finale is the highest-numbered episode of a season whose every episode has
// already aired (the whole season is out). Not gated by lead time — it fires
// on release, like episode_release.
func (r *NotificationRepository) SeasonFinaleCandidates(ctx context.Context, since time.Time) ([]EventCandidate, error) {
	var out []EventCandidate
	err := r.db.WithContext(ctx).Raw(`
		SELECT us.user_id           AS user_id,
		       e.show_id            AS show_id,
		       e.id                 AS episode_id,
		       s.title              AS show_title,
		       e.season_number      AS season_number,
		       e.episode_number     AS episode_number,
		       u.telegram_chat_id   AS telegram_chat_id,
		       u.apns_device_token  AS apns_token,
		       u.fcm_device_token   AS fcm_token
		FROM episodes e
		JOIN shows s       ON s.id = e.show_id
		JOIN user_shows us ON us.show_id = e.show_id AND us.status = 'watching'
		JOIN users u       ON u.id = us.user_id
		LEFT JOIN notification_prefs p ON p.user_id = us.user_id
		WHERE e.air_date IS NOT NULL
		  AND e.air_date <= now()::date
		  AND e.air_date >= ?
		  AND e.season_number > 0
		  AND COALESCE(p.season_finale, false) = true
		  AND (u.telegram_chat_id IS NOT NULL OR u.apns_device_token IS NOT NULL OR u.fcm_device_token IS NOT NULL)
		  AND NOT EXISTS (
		        SELECT 1 FROM episodes e2
		        WHERE e2.show_id = e.show_id AND e2.season_number = e.season_number
		          AND e2.episode_number > e.episode_number)
		  AND NOT EXISTS (
		        SELECT 1 FROM episodes e3
		        WHERE e3.show_id = e.show_id AND e3.season_number = e.season_number
		          AND (e3.air_date IS NULL OR e3.air_date > now()::date))`, since).Scan(&out).Error
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
