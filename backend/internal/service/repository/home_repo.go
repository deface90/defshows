package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/deface90/defshows/backend/internal/service/entity"
)

// HomeRepository serves the personal home: taste aggregation, catalog-based
// recommendations, and the per-user cache of that (heavy) computed part.
type HomeRepository struct {
	db *gorm.DB
}

// NewHomeRepository creates a HomeRepository.
func NewHomeRepository(db *gorm.DB) *HomeRepository {
	return &HomeRepository{db: db}
}

// GenreWeight is a genre with its accumulated taste weight.
type GenreWeight struct {
	ID     int64
	Name   string
	Weight int
}

// LangWeight is an original language with its accumulated taste weight.
type LangWeight struct {
	Code   string
	Weight int
}

// tasteWeight scores a tracked show by how much it signals preference: finished
// and actively-watched shows count most, dropped ones not at all.
const tasteWeight = "SUM(CASE us.status " +
	"WHEN 'completed' THEN 3 WHEN 'watching' THEN 2 WHEN 'dropped' THEN 0 ELSE 1 END)"

// TopGenres returns the user's most-watched genres by weighted tracking.
func (r *HomeRepository) TopGenres(ctx context.Context, userID int64, limit int) ([]GenreWeight, error) {
	var out []GenreWeight
	err := r.db.WithContext(ctx).
		Table("user_shows AS us").
		Select("g.id AS id, g.name AS name, "+tasteWeight+" AS weight").
		Joins("JOIN show_genres sg ON sg.show_id = us.show_id").
		Joins("JOIN genres g ON g.id = sg.genre_id").
		Where("us.user_id = ?", userID).
		Group("g.id, g.name").
		Having(tasteWeight + " > 0").
		Order("weight DESC, g.name ASC").
		Limit(limit).
		Scan(&out).Error
	return out, err
}

// TopLanguages returns the user's most-watched original languages.
func (r *HomeRepository) TopLanguages(ctx context.Context, userID int64, limit int) ([]LangWeight, error) {
	var out []LangWeight
	err := r.db.WithContext(ctx).
		Table("user_shows AS us").
		Select("s.original_language AS code, "+tasteWeight+" AS weight").
		Joins("JOIN shows s ON s.id = us.show_id").
		Where("us.user_id = ? AND s.original_language <> ''", userID).
		Group("s.original_language").
		Having(tasteWeight + " > 0").
		Order("weight DESC, code ASC").
		Limit(limit).
		Scan(&out).Error
	return out, err
}

// Recommendations returns catalog shows matching the given genres that the user
// does not already track, ranked by genre-overlap then popularity.
func (r *HomeRepository) Recommendations(ctx context.Context, userID int64, genreIDs []int64, limit int) ([]entity.Show, error) {
	if len(genreIDs) == 0 {
		return nil, nil
	}
	var shows []entity.Show
	err := r.db.WithContext(ctx).
		Model(&entity.Show{}).
		Select("shows.*, (SELECT COUNT(*) FROM seasons WHERE seasons.show_id = shows.id AND seasons.season_number > 0) AS season_count").
		Joins("JOIN show_genres sg ON sg.show_id = shows.id").
		Where("sg.genre_id IN ?", genreIDs).
		Where("shows.id NOT IN (SELECT show_id FROM user_shows WHERE user_id = ?)", userID).
		Group("shows.id").
		Order("COUNT(sg.genre_id) DESC, shows.popularity DESC").
		Limit(limit).
		Find(&shows).Error
	return shows, err
}

// GetHomeCache returns the cached home payload for a user or ErrNotFound.
func (r *HomeRepository) GetHomeCache(ctx context.Context, userID int64) (*entity.UserHomeCache, error) {
	var c entity.UserHomeCache
	err := r.db.WithContext(ctx).First(&c, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpsertHomeCache stores (or replaces) the cached home payload for a user.
func (r *HomeRepository) UpsertHomeCache(ctx context.Context, c *entity.UserHomeCache) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"payload", "computed_at"}),
		}).
		Create(c).Error
}
