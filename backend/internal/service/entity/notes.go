package entity

import "time"

// NoteScope categorizes a note/recap target.
type NoteScope string

const (
	ScopeShow    NoteScope = "show"
	ScopeSeason  NoteScope = "season"
	ScopeEpisode NoteScope = "episode"
)

// Note is a user's private note on a show/season/episode.
type Note struct {
	ID            int64 `gorm:"primaryKey"`
	UserID        int64
	ShowID        int64
	Scope         NoteScope
	SeasonNumber  *int
	EpisodeNumber *int
	Body          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TableName maps Note to the notes table.
func (Note) TableName() string { return "notes" }

// Recap is a public, generated summary of a show/season/episode.
type Recap struct {
	ID            int64 `gorm:"primaryKey"`
	ShowID        int64
	Scope         NoteScope
	SeasonNumber  *int
	EpisodeNumber *int
	Body          string
	Language      string
	Model         string
	GeneratedAt   time.Time
}

// TableName maps Recap to the recaps table.
func (Recap) TableName() string { return "recaps" }
