package entity

import "time"

// FollowStatus is the state of a follow relationship.
type FollowStatus string

const (
	// FollowPending is a follow request awaiting the followee's approval
	// (only private profiles produce pending follows).
	FollowPending FollowStatus = "pending"
	// FollowAccepted grants the follower access to the followee's collection and
	// activity, and subscribes the follower to the followee's events in the feed.
	FollowAccepted FollowStatus = "accepted"
)

// Follow is a directed follow edge: FollowerID follows FolloweeID.
type Follow struct {
	ID         int64 `gorm:"primaryKey"`
	FollowerID int64
	FolloweeID int64
	Status     FollowStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TableName maps Follow to the follows table.
func (Follow) TableName() string { return "follows" }

// EventType enumerates the kinds of activity events recorded for the feed.
type EventType string

const (
	EventWatchedEpisode EventType = "watched_episode"
	EventFinishedSeason EventType = "finished_season"
	EventFinishedShow   EventType = "finished_show"
	EventAddedShow      EventType = "added_show"
	EventRatedShow      EventType = "rated_show"
)

// ActivityEvent is an append-only record of a user action, read back as the
// profile and home activity feeds. SeasonNumber/EpisodeID/Rating are set only
// for the event types that carry them.
type ActivityEvent struct {
	ID           int64 `gorm:"primaryKey"`
	UserID       int64
	Type         EventType
	ShowID       int64
	SeasonNumber *int
	EpisodeID    *int64
	Rating       *int
	CreatedAt    time.Time
}

// TableName maps ActivityEvent to the activity_events table.
func (ActivityEvent) TableName() string { return "activity_events" }

// Block is a directed block edge: BlockerID has blocked BlockedID. Treated as
// symmetric for visibility (either direction hides the pair from each other)
// while the directed row records who blocked whom (only the blocker can unblock).
type Block struct {
	ID        int64 `gorm:"primaryKey"`
	BlockerID int64
	BlockedID int64
	CreatedAt time.Time
}

// TableName maps Block to the blocks table.
func (Block) TableName() string { return "blocks" }
