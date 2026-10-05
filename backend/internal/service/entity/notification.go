package entity

import "time"

// Notification types.
const (
	NotifyEpisodeReleased = "episode_released"
	NotifyEpisodeUpcoming = "episode_upcoming"
	NotifySeasonUpcoming  = "season_upcoming"
	NotifySeasonFinale    = "season_finale"
	NotifyFollowRequest   = "follow_request"
	NotifyFollowAccepted  = "follow_accepted"
	NotifyFollowNew       = "follow_new"
)

// Delivery statuses (per NotificationDelivery row).
const (
	NotifyPending = "pending"
	NotifySent    = "sent"
	NotifyFailed  = "failed"
)

// NotificationPref holds a user's notification preferences.
type NotificationPref struct {
	ID             int64 `gorm:"primaryKey"`
	UserID         int64
	EpisodeRelease bool
	SeasonStart    bool
	SeasonFinale   bool
	WeeklyDigest   bool
	SocialFollows  bool
	Channel        string
	LeadTimeHours  int
}

// TableName maps NotificationPref to the notification_prefs table.
func (NotificationPref) TableName() string { return "notification_prefs" }

// DefaultNotificationPref returns the default prefs for a user.
func DefaultNotificationPref(userID int64) NotificationPref {
	return NotificationPref{
		UserID:         userID,
		EpisodeRelease: false,
		SeasonStart:    true,
		SeasonFinale:   false,
		WeeklyDigest:   false,
		SocialFollows:  true,
		Channel:        "telegram",
		LeadTimeHours:  24,
	}
}

// Notification is one logical event and the in-app feed entry (shown once in the feed,
// regardless of how many channels deliver it). Per-channel outbox delivery is tracked
// separately in NotificationDelivery.
type Notification struct {
	ID        int64 `gorm:"primaryKey"`
	UserID    int64
	Type      string
	ShowID    *int64
	EpisodeID *int64
	// ActorID references the user who triggered the notification (e.g. the follower
	// for follow_request/follow_new/follow_accepted); nil for catalog/episode notifications.
	ActorID   *int64
	DedupeKey string
	Payload   string
	ReadAt    *time.Time
	CreatedAt time.Time
}

// TableName maps Notification to the notifications table.
func (Notification) TableName() string { return "notifications" }

// NotificationDelivery is a per-channel outbox entry for a Notification. The sender polls
// pending rows by channel and marks them sent/failed. A notification with no deliveries is
// feed-only (e.g. a follow notification for a user with no linked Telegram/push channel).
type NotificationDelivery struct {
	ID             int64 `gorm:"primaryKey"`
	NotificationID int64
	Channel        string
	Status         string
	ScheduledFor   time.Time
	SentAt         *time.Time
}

// TableName maps NotificationDelivery to the notification_deliveries table.
func (NotificationDelivery) TableName() string { return "notification_deliveries" }

// TelegramLinkToken is a one-time token linking a Telegram chat to a user.
type TelegramLinkToken struct {
	Token     string `gorm:"primaryKey"`
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// TableName maps TelegramLinkToken to the telegram_link_tokens table.
func (TelegramLinkToken) TableName() string { return "telegram_link_tokens" }
