package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/deface90/defshows/backend/internal/service/entity"
)

// NotesRepository is the data access layer for notes and recaps.
type NotesRepository struct {
	db *gorm.DB
}

// NewNotesRepository creates a NotesRepository.
func NewNotesRepository(db *gorm.DB) *NotesRepository {
	return &NotesRepository{db: db}
}

// CreateNote inserts a private note.
func (r *NotesRepository) CreateNote(ctx context.Context, n *entity.Note) error {
	return r.db.WithContext(ctx).Create(n).Error
}

// GetNote returns a note by id, scoped to its owner, or ErrNotFound.
func (r *NotesRepository) GetNote(ctx context.Context, userID, noteID int64) (*entity.Note, error) {
	var n entity.Note
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", noteID, userID).First(&n).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// UpdateNoteBody updates a note's body if owned by the user. Returns ErrNotFound
// if no such owned note exists.
func (r *NotesRepository) UpdateNoteBody(ctx context.Context, userID, noteID int64, body string) error {
	res := r.db.WithContext(ctx).Model(&entity.Note{}).
		Where("id = ? AND user_id = ?", noteID, userID).
		Updates(map[string]any{"body": body, "updated_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteNote deletes a note owned by the user.
func (r *NotesRepository) DeleteNote(ctx context.Context, userID, noteID int64) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", noteID, userID).
		Delete(&entity.Note{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListNotes lists a user's notes for a show.
func (r *NotesRepository) ListNotes(ctx context.Context, userID, showID int64) ([]entity.Note, error) {
	var notes []entity.Note
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND show_id = ?", userID, showID).
		Order("created_at").Find(&notes).Error
	return notes, err
}

// GetRecap returns a public recap or ErrNotFound.
func (r *NotesRepository) GetRecap(ctx context.Context, showID int64, scope string, season, episode *int, language string) (*entity.Recap, error) {
	q := r.db.WithContext(ctx).
		Where("show_id = ? AND scope = ? AND language = ?", showID, scope, language)
	if season == nil {
		q = q.Where("season_number IS NULL")
	} else {
		q = q.Where("season_number = ?", *season)
	}
	if episode == nil {
		q = q.Where("episode_number IS NULL")
	} else {
		q = q.Where("episode_number = ?", *episode)
	}
	var rec entity.Recap
	err := q.First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// UpsertRecap stores/replaces a generated recap.
func (r *NotesRepository) UpsertRecap(ctx context.Context, rec *entity.Recap) error {
	// The unique index uses COALESCE on nullable columns, which gorm's ON
	// CONFLICT cannot target by column list; do an explicit find-or-update.
	existing, err := r.GetRecap(ctx, rec.ShowID, string(rec.Scope), rec.SeasonNumber, rec.EpisodeNumber, rec.Language)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if existing != nil {
		rec.ID = existing.ID
		return r.db.WithContext(ctx).Model(&entity.Recap{}).Where("id = ?", existing.ID).
			Updates(map[string]any{"body": rec.Body, "model": rec.Model, "generated_at": time.Now()}).Error
	}
	return r.db.WithContext(ctx).Create(rec).Error
}
