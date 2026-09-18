package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/deface90/defshows/backend/internal/service/entity"
)

// TrackingRepository is the data access layer for user tracking.
type TrackingRepository struct {
	db *gorm.DB
}

// NewTrackingRepository creates a TrackingRepository.
func NewTrackingRepository(db *gorm.DB) *TrackingRepository {
	return &TrackingRepository{db: db}
}

// AddUserShow inserts a tracked show. Returns ErrConflict if already tracked.
func (r *TrackingRepository) AddUserShow(ctx context.Context, us *entity.UserShow) error {
	if err := r.db.WithContext(ctx).Create(us).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrConflict
		}
		return err
	}
	return nil
}

// GetUserShow returns a user's tracked show by (userID, showID) or ErrNotFound.
func (r *TrackingRepository) GetUserShow(ctx context.Context, userID, showID int64) (*entity.UserShow, error) {
	var us entity.UserShow
	err := r.db.WithContext(ctx).Where("user_id = ? AND show_id = ?", userID, showID).First(&us).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &us, nil
}

// ListUserShows returns a user's tracked shows, optionally filtered by status.
func (r *TrackingRepository) ListUserShows(ctx context.Context, userID int64, status string) ([]entity.UserShow, error) {
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var list []entity.UserShow
	err := q.Order("added_at DESC").Find(&list).Error
	return list, err
}

// CountByUsers returns the number of tracked shows per user, keyed by user id,
// in a single grouped query (users with none are absent from the map).
func (r *TrackingRepository) CountByUsers(ctx context.Context) (map[int64]int, error) {
	type row struct {
		UserID int64
		Count  int
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Model(&entity.UserShow{}).
		Select("user_id, count(*) as count").
		Group("user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int64]int, len(rows))
	for _, rw := range rows {
		out[rw.UserID] = rw.Count
	}
	return out, nil
}

// UpdateUserShow persists status/favorite/preferred_dubbing for a user show.
func (r *TrackingRepository) UpdateUserShow(ctx context.Context, us *entity.UserShow) error {
	return r.db.WithContext(ctx).Model(&entity.UserShow{}).
		Where("id = ?", us.ID).
		Updates(map[string]any{
			"status":            us.Status,
			"favorite":          us.Favorite,
			"preferred_dubbing": us.PreferredDubbing,
			"updated_at":        gorm.Expr("now()"),
		}).Error
}

// RemoveUserShow deletes a user's tracked show.
func (r *TrackingRepository) RemoveUserShow(ctx context.Context, userID, showID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND show_id = ?", userID, showID).
		Delete(&entity.UserShow{}).Error
}

// UpsertUserEpisode marks/updates a per-episode watch record.
func (r *TrackingRepository) UpsertUserEpisode(ctx context.Context, ue *entity.UserEpisode) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_show_id"}, {Name: "episode_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"watched", "watched_at", "rating"}),
	}).Create(ue).Error
}

// WatchedEpisodeIDs returns the ids of watched episodes for a user show.
func (r *TrackingRepository) WatchedEpisodeIDs(ctx context.Context, userShowID int64) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&entity.UserEpisode{}).
		Where("user_show_id = ? AND watched = true", userShowID).
		Pluck("episode_id", &ids).Error
	return ids, err
}

// AddLink adds a user's reference link.
func (r *TrackingRepository) AddLink(ctx context.Context, link *entity.UserShowLink) error {
	return r.db.WithContext(ctx).Create(link).Error
}

// ListLinks returns a user show's links.
func (r *TrackingRepository) ListLinks(ctx context.Context, userShowID int64) ([]entity.UserShowLink, error) {
	var links []entity.UserShowLink
	err := r.db.WithContext(ctx).Where("user_show_id = ?", userShowID).Order("id").Find(&links).Error
	return links, err
}

// DeleteLink removes a link owned by the given user show.
func (r *TrackingRepository) DeleteLink(ctx context.Context, userShowID, linkID int64) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_show_id = ?", linkID, userShowID).
		Delete(&entity.UserShowLink{}).Error
}

// WatchShow marks all currently catalogued episodes and completes the user's show
// atomically. Existing watch dates and ratings are preserved on repeated calls.
func (r *TrackingRepository) WatchShow(ctx context.Context, userID, showID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&entity.UserShow{}).Where("user_id = ? AND show_id = ?", userID, showID).
			Updates(map[string]any{"status": entity.StatusCompleted, "updated_at": gorm.Expr("now()")})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return tx.Exec(`
   INSERT INTO user_episodes (user_show_id, episode_id, watched, watched_at)
   SELECT us.id, e.id, true, now()
   FROM user_shows us JOIN episodes e ON e.show_id = us.show_id
   WHERE us.user_id = ? AND us.show_id = ?
   ON CONFLICT (user_show_id, episode_id) DO UPDATE
   SET watched = true, watched_at = EXCLUDED.watched_at
   WHERE user_episodes.watched = false
  `, userID, showID).Error
	})
}

// UpdateLink updates only the value of a link owned by the given user show.
func (r *TrackingRepository) UpdateLink(ctx context.Context, userShowID, linkID int64, value string) error {
	result := r.db.WithContext(ctx).Model(&entity.UserShowLink{}).
		Where("id = ? AND user_show_id = ?", linkID, userShowID).Update("url", value)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
