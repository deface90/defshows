package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/deface90/defshows/backend/internal/service/entity"
)

// FeedCursor is a keyset position in a feed: the (created_at, id) of the last raw
// event consumed. Order is (created_at DESC, id DESC), so a page fetches rows
// strictly "before" the cursor.
type FeedCursor struct {
	CreatedAt time.Time
	ID        int64
}

// ActivityRepository is the data access layer for the append-only activity feed.
// Feed emission on the tracking hot path happens inside the tracking repo's own
// transactions (see insertActivityEvents); this type exposes a standalone write
// path used off the hot path (e.g. the history backfill in cmd/backfill-activity).
type ActivityRepository struct {
	db *gorm.DB
}

// NewActivityRepository creates an ActivityRepository.
func NewActivityRepository(db *gorm.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// InsertEvents appends activity events in a single statement. CreatedAt is filled
// by gorm when zero.
func (r *ActivityRepository) InsertEvents(ctx context.Context, events ...entity.ActivityEvent) error {
	return insertActivityEvents(r.db.WithContext(ctx), events...)
}

// ProfileEvents returns a page of a single user's raw events, newest first, after
// the given cursor (nil = first page).
func (r *ActivityRepository) ProfileEvents(ctx context.Context, userID int64, cursor *FeedCursor, limit int) ([]entity.ActivityEvent, error) {
	return pageEvents(r.db.WithContext(ctx).Where("user_id = ?", userID), cursor, limit)
}

// HomeEvents returns a page of raw events across the given users (the viewer's
// accepted followees), newest first, after the given cursor. Empty user set = no rows.
func (r *ActivityRepository) HomeEvents(ctx context.Context, userIDs []int64, cursor *FeedCursor, limit int) ([]entity.ActivityEvent, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	return pageEvents(r.db.WithContext(ctx).Where("user_id IN ?", userIDs), cursor, limit)
}

func pageEvents(q *gorm.DB, cursor *FeedCursor, limit int) ([]entity.ActivityEvent, error) {
	if cursor != nil {
		// Row-value comparison is index-friendly for keyset pagination.
		q = q.Where("(created_at, id) < (?, ?)", cursor.CreatedAt, cursor.ID)
	}
	var events []entity.ActivityEvent
	err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&events).Error
	return events, err
}

// insertActivityEvents appends activity events using the given handle (typically a
// transaction shared with the tracking mutation that produced them, so the feed can
// never drift from the underlying tracking state). A no-op when there are no events.
func insertActivityEvents(tx *gorm.DB, events ...entity.ActivityEvent) error {
	if len(events) == 0 {
		return nil
	}
	return tx.Create(&events).Error
}
