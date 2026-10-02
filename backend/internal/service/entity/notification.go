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
)

// ChannelInApp is a delivery channel with no outbox sender: notifications on this
// channel are surfaced only in the in-app feed (follow_request/follow_accepted).
// It is deliberately absent from the sender's pendingQueries, so nothing dispatches it.
const ChannelInApp = "in_app"

// Notification statuses.
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
	Channel        string
	LeadTimeHours  int
}

// TableName maps NotificationPref to the notification_prefs table.
func (NotificationPref) TableName() string { return "notification_prefs" }

// DefaultNotificationPref returns the default prefs for a user.
func DefaultNotificationPref(userID int64) NotificationPref {
	return NotificationPref{
		UserID:         userID,
		EpisodeRelease: true,
		SeasonStart:    true,
		SeasonFinale:   true,
		WeeklyDigest:   false,
		Channel:        "telegram",
		LeadTimeHours:  24,
	}
}

// Notification is an outbox entry that also serves as the in-app feed.
type Notification struct {
	ID        int64 `gorm:"primaryKey"`
	UserID    int64
	Type      string
	ShowID    *int64
	EpisodeID *int64
	// ActorID references the user who triggered the notification (e.g. the follower
	// for follow_request/follow_accepted); nil for catalog/episode notifications.
	ActorID      *int64
	Channel      string
	Status       string
	ScheduledFor time.Time
	SentAt       *time.Time
	DedupeKey    string
	Payload      string
	ReadAt       *time.Time
	CreatedAt    time.Time
}

// TableName maps Notification to the notifications table.
func (Notification) TableName() string { return "notifications" }

// TelegramLinkToken is a one-time token linking a Telegram chat to a user.
type TelegramLinkToken struct {
	Token     string `gorm:"primaryKey"`
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// TableName maps TelegramLinkToken to the telegram_link_tokens table.
func (TelegramLinkToken) TableName() string { return "telegram_link_tokens" }
