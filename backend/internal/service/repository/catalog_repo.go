package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/deface90/defshows/backend/internal/service/entity"
)

// CatalogRepository is the data access layer for the mirrored TMDB catalog.
type CatalogRepository struct {
	db *gorm.DB
}

// NewCatalogRepository creates a CatalogRepository.
func NewCatalogRepository(db *gorm.DB) *CatalogRepository {
	return &CatalogRepository{db: db}
}

// UpsertShow inserts or updates a show by tmdb_id. next_episode_id is managed
// separately via SetNextEpisode (episodes must exist first).
func (r *CatalogRepository) UpsertShow(ctx context.Context, s *entity.Show) error {
	if s.AiringStatus == "" {
		s.AiringStatus = entity.AiringNotStarted
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tmdb_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "original_title", "overview", "poster_key", "backdrop_key",
			"status", "in_production", "first_air_date", "last_air_date",
			"next_episode_air_date", "last_episode_air_date", "airing_status",
			"original_language", "popularity", "vote_average", "vote_count",
			"last_synced_at", "updated_at",
		}),
	}).Create(s).Error
}

// GetShowByID returns a show by internal id or ErrNotFound.
func (r *CatalogRepository) GetShowByID(ctx context.Context, id int64) (*entity.Show, error) {
	var s entity.Show
	err := r.db.WithContext(ctx).First(&s, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetShowByTMDBID returns a show by tmdb_id or ErrNotFound.
func (r *CatalogRepository) GetShowByTMDBID(ctx context.Context, tmdbID int64) (*entity.Show, error) {
	var s entity.Show
	err := r.db.WithContext(ctx).Where("tmdb_id = ?", tmdbID).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetShowWithRelations returns a show with seasons and genres preloaded.
func (r *CatalogRepository) GetShowWithRelations(ctx context.Context, id int64) (*entity.Show, error) {
	var s entity.Show
	err := r.db.WithContext(ctx).
		Preload("Seasons", func(db *gorm.DB) *gorm.DB { return db.Order("season_number") }).
		Preload("Genres").
		First(&s, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpsertSeasons upserts seasons by (show_id, season_number).
func (r *CatalogRepository) UpsertSeasons(ctx context.Context, seasons []entity.Season) error {
	if len(seasons) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "show_id"}, {Name: "season_number"}},
		UpdateAll: true,
	}).Create(&seasons).Error
}

// GetSeasonsByShow returns a show's seasons ordered by number.
func (r *CatalogRepository) GetSeasonsByShow(ctx context.Context, showID int64) ([]entity.Season, error) {
	var seasons []entity.Season
	err := r.db.WithContext(ctx).Where("show_id = ?", showID).Order("season_number").Find(&seasons).Error
	return seasons, err
}

// UpsertEpisodes upserts episodes by (show_id, season_number, episode_number).
func (r *CatalogRepository) UpsertEpisodes(ctx context.Context, episodes []entity.Episode) error {
	if len(episodes) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "show_id"}, {Name: "season_number"}, {Name: "episode_number"}},
		UpdateAll: true,
	}).Create(&episodes).Error
}

// GetEpisodesByShow returns a show's episodes ordered by season/episode.
func (r *CatalogRepository) GetEpisodesByShow(ctx context.Context, showID int64) ([]entity.Episode, error) {
	var episodes []entity.Episode
	err := r.db.WithContext(ctx).
		Where("show_id = ?", showID).
		Order("season_number, episode_number").
		Find(&episodes).Error
	return episodes, err
}

// SetNextEpisode updates the show's next-episode pointer and derived fields.
func (r *CatalogRepository) SetNextEpisode(ctx context.Context, showID int64, episodeID *int64, airDate *time.Time, status entity.AiringStatus) error {
	return r.db.WithContext(ctx).Model(&entity.Show{}).Where("id = ?", showID).Updates(map[string]any{
		"next_episode_id":       episodeID,
		"next_episode_air_date": airDate,
		"airing_status":         status,
		"updated_at":            time.Now(),
	}).Error
}

// UpsertGenres upserts genres by tmdb_id, populating their ids.
func (r *CatalogRepository) UpsertGenres(ctx context.Context, genres []entity.Genre) error {
	if len(genres) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tmdb_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name"}),
	}).Create(&genres).Error
}

// ReplaceShowGenres sets the show's genre associations.
func (r *CatalogRepository) ReplaceShowGenres(ctx context.Context, showID int64, genreIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("show_id = ?", showID).Delete(&showGenre{}).Error; err != nil {
			return err
		}
		if len(genreIDs) == 0 {
			return nil
		}
		links := make([]showGenre, 0, len(genreIDs))
		for _, gid := range genreIDs {
			links = append(links, showGenre{ShowID: showID, GenreID: gid})
		}
		return tx.Create(&links).Error
	})
}

// ListShows returns local catalog shows matching query (empty = all), ordered by
// popularity.
func (r *CatalogRepository) ListShows(ctx context.Context, query string, limit, offset int) ([]entity.Show, error) {
	q := r.db.WithContext(ctx).Model(&entity.Show{})
	if query != "" {
		like := "%" + query + "%"
		q = q.Where("title ILIKE ? OR original_title ILIKE ?", like, like)
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var shows []entity.Show
	err := q.Order("popularity DESC").Limit(limit).Offset(offset).Find(&shows).Error
	return shows, err
}

// UpsertRatings upserts show ratings by (show_id, source).
func (r *CatalogRepository) UpsertRatings(ctx context.Context, ratings []entity.ShowRating) error {
	if len(ratings) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "show_id"}, {Name: "source"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "votes", "fetched_at"}),
	}).Create(&ratings).Error
}

// GetRatingsByShow returns a show's ratings.
func (r *CatalogRepository) GetRatingsByShow(ctx context.Context, showID int64) ([]entity.ShowRating, error) {
	var ratings []entity.ShowRating
	err := r.db.WithContext(ctx).Where("show_id = ?", showID).Order("source").Find(&ratings).Error
	return ratings, err
}

// ShowsForRefresh returns shows that are candidates for a catalog refresh:
// in-production or currently/between airing, or not synced since staleBefore.
// Oldest sync first.
func (r *CatalogRepository) ShowsForRefresh(ctx context.Context, staleBefore time.Time, limit int) ([]entity.Show, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var shows []entity.Show
	err := r.db.WithContext(ctx).
		Where(`in_production = true
			OR airing_status IN ('airing', 'between_seasons')
			OR last_synced_at IS NULL
			OR last_synced_at < ?`, staleBefore).
		Order("last_synced_at ASC NULLS FIRST").
		Limit(limit).
		Find(&shows).Error
	return shows, err
}

// showGenre is the join table row (internal to the repository).
type showGenre struct {
	ShowID  int64
	GenreID int64
}

func (showGenre) TableName() string { return "show_genres" }
