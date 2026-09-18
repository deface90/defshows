package usecase

import (
	"context"
	"errors"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
)

// Notes usecase errors.
var (
	ErrNoteNotFound      = errors.New("usecase: note not found")
	ErrRecapNotAvailable = errors.New("usecase: recap not available")
)

// NotesRepo is the storage dependency of NotesUsecase.
type NotesRepo interface {
	CreateNote(ctx context.Context, n *entity.Note) error
	GetNote(ctx context.Context, userID, noteID int64) (*entity.Note, error)
	UpdateNoteBody(ctx context.Context, userID, noteID int64, body string) error
	DeleteNote(ctx context.Context, userID, noteID int64) error
	ListNotes(ctx context.Context, userID, showID int64) ([]entity.Note, error)
	GetRecap(ctx context.Context, showID int64, scope string, season, episode *int, language string) (*entity.Recap, error)
}

// NotesUsecase implements private notes and public recap reads.
type NotesUsecase struct {
	repo NotesRepo
}

// NewNotesUsecase creates a NotesUsecase.
func NewNotesUsecase(repo NotesRepo) *NotesUsecase {
	return &NotesUsecase{repo: repo}
}

// CreateNote adds a private note.
func (uc *NotesUsecase) CreateNote(ctx context.Context, userID, showID int64, scope string, season, episode *int, body string) (*entity.Note, error) {
	n := &entity.Note{
		UserID:        userID,
		ShowID:        showID,
		Scope:         entity.NoteScope(scope),
		SeasonNumber:  season,
		EpisodeNumber: episode,
		Body:          body,
	}
	if err := uc.repo.CreateNote(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

// UpdateNote updates a note's body (owner only) and returns the fresh note.
func (uc *NotesUsecase) UpdateNote(ctx context.Context, userID, noteID int64, body string) (*entity.Note, error) {
	if err := uc.repo.UpdateNoteBody(ctx, userID, noteID, body); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, err
	}
	return uc.repo.GetNote(ctx, userID, noteID)
}

// DeleteNote deletes a note (owner only).
func (uc *NotesUsecase) DeleteNote(ctx context.Context, userID, noteID int64) error {
	if err := uc.repo.DeleteNote(ctx, userID, noteID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNoteNotFound
		}
		return err
	}
	return nil
}

// ListNotes lists a user's notes for a show.
func (uc *NotesUsecase) ListNotes(ctx context.Context, userID, showID int64) ([]entity.Note, error) {
	return uc.repo.ListNotes(ctx, userID, showID)
}

// GetRecap returns a public recap, or ErrRecapNotAvailable if not yet generated.
func (uc *NotesUsecase) GetRecap(ctx context.Context, showID int64, scope string, season, episode *int, language string) (*entity.Recap, error) {
	if language == "" {
		language = "ru"
	}
	rec, err := uc.repo.GetRecap(ctx, showID, scope, season, episode, language)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecapNotAvailable
		}
		return nil, err
	}
	return rec, nil
}
