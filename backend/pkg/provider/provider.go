// Package provider defines the external catalog/ratings provider interfaces and
// their domain-neutral DTOs, so implementations (TMDB, OMDb, ...) can be swapped.
package provider

import (
	"context"
	"time"
)

// ShowSummary is a lightweight search result.
type ShowSummary struct {
	TMDBID        int64
	Title         string
	OriginalTitle string
	Overview      string
	PosterURL     string
	FirstAirDate  *time.Time
	Popularity    float64
	VoteAverage   float64
	VoteCount     int64
}

// Show is full catalog detail for a series.
type Show struct {
	TMDBID           int64
	Title            string
	OriginalTitle    string
	Overview         string
	PosterURL        string
	BackdropURL      string
	Status           string
	InProduction     bool
	FirstAirDate     *time.Time
	LastAirDate      *time.Time
	OriginalLanguage string
	Popularity       float64
	VoteAverage      float64
	VoteCount        int64
	Genres           []Genre
	Seasons          []Season
	NextEpisode      *Episode
	LastEpisode      *Episode
}

// Genre is a catalog genre.
type Genre struct {
	TMDBID int64
	Name   string
}

// Season is a season with optional episodes (populated by GetSeason).
type Season struct {
	TMDBID       *int64
	SeasonNumber int
	Name         string
	Overview     string
	AirDate      *time.Time
	EpisodeCount int
	PosterURL    string
	Episodes     []Episode
}

// Episode is a single episode.
type Episode struct {
	TMDBID        *int64
	SeasonNumber  int
	EpisodeNumber int
	Name          string
	Overview      string
	AirDate       *time.Time
	Runtime       *int
	StillURL      string
}

// ShowProvider is the catalog source (e.g. TMDB).
type ShowProvider interface {
	SearchShows(ctx context.Context, query string) ([]ShowSummary, error)
	GetShow(ctx context.Context, tmdbID int64) (*Show, error)
	GetSeason(ctx context.Context, tmdbID int64, seasonNumber int) (*Season, error)
}

// Rating is a single normalized rating from a source.
type Rating struct {
	Source string // imdb | rotten_tomatoes | metacritic | kinopoisk | tmdb
	Value  string // as reported, e.g. "8.5/10", "90%"
	Votes  *int64
}

// RatingProvider is a ratings aggregator source (e.g. OMDb).
type RatingProvider interface {
	Ratings(ctx context.Context, title string, year int) ([]Rating, error)
}
