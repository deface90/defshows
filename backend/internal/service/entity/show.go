package entity

import "time"

// AiringStatus is the derived airing state of a show, recomputed on sync.
type AiringStatus string

const (
	AiringNotStarted     AiringStatus = "not_started"
	AiringNow            AiringStatus = "airing"
	AiringBetweenSeasons AiringStatus = "between_seasons"
	AiringEnded          AiringStatus = "ended"
)

// Show is a catalog entry mirrored from TMDB.
type Show struct {
	ID                 int64 `gorm:"primaryKey"`
	TMDBID             int64 `gorm:"column:tmdb_id"`
	Title              string
	OriginalTitle      string
	Overview           string
	PosterKey          *string
	BackdropKey        *string
	Status             string
	InProduction       bool
	FirstAirDate       *time.Time
	LastAirDate        *time.Time
	NextEpisodeID      *int64
	NextEpisodeAirDate *time.Time
	LastEpisodeAirDate *time.Time
	AiringStatus       AiringStatus
	OriginalLanguage   string
	Popularity         float64
	VoteAverage        float64
	VoteCount          int64
	LastSyncedAt       *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time

	Seasons []Season `gorm:"foreignKey:ShowID"`
	Genres  []Genre  `gorm:"many2many:show_genres;"`
}

// TableName maps Show to the shows table.
func (Show) TableName() string { return "shows" }

// Season is a show season.
type Season struct {
	ID           int64 `gorm:"primaryKey"`
	ShowID       int64
	TMDBID       *int64 `gorm:"column:tmdb_id"`
	SeasonNumber int
	Name         string
	Overview     string
	AirDate      *time.Time
	EpisodeCount int
	PosterKey    *string
}

// TableName maps Season to the seasons table.
func (Season) TableName() string { return "seasons" }

// Episode is a single episode.
type Episode struct {
	ID            int64 `gorm:"primaryKey"`
	SeasonID      int64
	ShowID        int64
	TMDBID        *int64 `gorm:"column:tmdb_id"`
	SeasonNumber  int
	EpisodeNumber int
	Name          string
	Overview      string
	AirDate       *time.Time
	Runtime       *int
	StillKey      *string
}

// TableName maps Episode to the episodes table.
func (Episode) TableName() string { return "episodes" }

// ShowRating is one rating source for a show (aggregator row).
type ShowRating struct {
	ID        int64 `gorm:"primaryKey"`
	ShowID    int64
	Source    string
	Value     string
	Votes     *int64
	FetchedAt time.Time
}

// TableName maps ShowRating to the show_ratings table.
func (ShowRating) TableName() string { return "show_ratings" }

// Genre is a TMDB genre.
type Genre struct {
	ID     int64 `gorm:"primaryKey"`
	TMDBID int64 `gorm:"column:tmdb_id"`
	Name   string
}

// TableName maps Genre to the genres table.
func (Genre) TableName() string { return "genres" }
