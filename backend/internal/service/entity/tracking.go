package entity

import "time"

// WatchStatus is a user's tracking status for a show.
type WatchStatus string

const (
	StatusWatching    WatchStatus = "watching"
	StatusPlanToWatch WatchStatus = "plan_to_watch"
	StatusOnHold      WatchStatus = "on_hold"
	StatusCompleted   WatchStatus = "completed"
	StatusDropped     WatchStatus = "dropped"
)

// UserShow is a show a user tracks.
type UserShow struct {
	ID               int64 `gorm:"primaryKey"`
	UserID           int64
	ShowID           int64
	Status           WatchStatus
	Favorite         bool
	PreferredDubbing string
	AddedAt          time.Time
	UpdatedAt        time.Time
}

// TableName maps UserShow to the user_shows table.
func (UserShow) TableName() string { return "user_shows" }

// UserEpisode is a per-episode watch mark.
type UserEpisode struct {
	ID         int64 `gorm:"primaryKey"`
	UserShowID int64
	EpisodeID  int64
	Watched    bool
	WatchedAt  *time.Time
	Rating     *int
}

// TableName maps UserEpisode to the user_episodes table.
func (UserEpisode) TableName() string { return "user_episodes" }

// LinkKind categorizes a user's reference link.
type LinkKind string

const (
	LinkDownload  LinkKind = "download"
	LinkStreaming LinkKind = "streaming"
	LinkWiki      LinkKind = "wiki"
	LinkIMDB      LinkKind = "imdb"
	LinkKinopoisk LinkKind = "kinopoisk"
)

// UserShowLink is a user's manual reference (download/streaming/wiki/imdb/kinopoisk).
type UserShowLink struct {
	ID         int64 `gorm:"primaryKey"`
	UserShowID int64
	Kind       LinkKind
	Label      string
	URL        string `gorm:"column:url"`
}

// TableName maps UserShowLink to the user_show_links table.
func (UserShowLink) TableName() string { return "user_show_links" }

// UserHomeCache stores the pre-computed heavy part of a user's home (taste
// profile + recommendations) as JSON, refreshed on demand past a TTL.
type UserHomeCache struct {
	UserID     int64  `gorm:"primaryKey"`
	Payload    []byte `gorm:"type:jsonb"`
	ComputedAt time.Time
}

// TableName maps UserHomeCache to the user_home_cache table.
func (UserHomeCache) TableName() string { return "user_home_cache" }

// DubbingStudio is an admin-managed dubbing reference.
type DubbingStudio struct {
	ID      int64 `gorm:"primaryKey"`
	Name    string
	SiteURL string `gorm:"column:site_url"`
	Active  bool
}

// TableName maps DubbingStudio to the dubbing_studios table.
func (DubbingStudio) TableName() string { return "dubbing_studios" }
