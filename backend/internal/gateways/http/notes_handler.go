package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	notesapi "github.com/deface90/defshows/backend/pkg/server/notes"
)

// NotesHandler implements the generated notes ServerInterface.
type NotesHandler struct {
	uc *usecase.NotesUsecase
}

// NewNotesHandler creates a NotesHandler.
func NewNotesHandler(uc *usecase.NotesUsecase) *NotesHandler {
	return &NotesHandler{uc: uc}
}

var _ notesapi.ServerInterface = (*NotesHandler)(nil)

// ListNotes handles GET /me/notes.
func (h *NotesHandler) ListNotes(c echo.Context, params notesapi.ListNotesParams) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	notes, err := h.uc.ListNotes(c.Request().Context(), uid, params.ShowId)
	if err != nil {
		return err
	}
	out := make([]notesapi.Note, 0, len(notes))
	for i := range notes {
		out = append(out, toAPINote(&notes[i]))
	}
	return c.JSON(http.StatusOK, notesapi.NoteList{Notes: out})
}

// CreateNote handles POST /me/notes.
func (h *NotesHandler) CreateNote(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	var req notesapi.CreateNoteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	note, err := h.uc.CreateNote(c.Request().Context(), uid, req.ShowId, string(req.Scope), req.SeasonNumber, req.EpisodeNumber, req.Body)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toAPINote(note))
}

// UpdateNote handles PATCH /me/notes/{noteId}.
func (h *NotesHandler) UpdateNote(c echo.Context, noteID int64) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	var req notesapi.UpdateNoteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	note, err := h.uc.UpdateNote(c.Request().Context(), uid, noteID, req.Body)
	if err != nil {
		if errors.Is(err, usecase.ErrNoteNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "note not found")
		}
		return err
	}
	return c.JSON(http.StatusOK, toAPINote(note))
}

// DeleteNote handles DELETE /me/notes/{noteId}.
func (h *NotesHandler) DeleteNote(c echo.Context, noteID int64) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.uc.DeleteNote(c.Request().Context(), uid, noteID); err != nil {
		if errors.Is(err, usecase.ErrNoteNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "note not found")
		}
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// GetRecap handles GET /shows/{showId}/recap.
func (h *NotesHandler) GetRecap(c echo.Context, showID int64, params notesapi.GetRecapParams) error {
	scope := "show"
	if params.Scope != nil {
		scope = string(*params.Scope)
	}
	lang := ""
	if params.Language != nil {
		lang = *params.Language
	}
	rec, err := h.uc.GetRecap(c.Request().Context(), showID, scope, params.SeasonNumber, params.EpisodeNumber, lang)
	if err != nil {
		if errors.Is(err, usecase.ErrRecapNotAvailable) {
			return echo.NewHTTPError(http.StatusNotFound, "recap not available")
		}
		return err
	}
	gen := rec.GeneratedAt.Format(time.RFC3339)
	return c.JSON(http.StatusOK, notesapi.Recap{
		ShowId:        rec.ShowID,
		Scope:         string(rec.Scope),
		SeasonNumber:  rec.SeasonNumber,
		EpisodeNumber: rec.EpisodeNumber,
		Body:          rec.Body,
		Language:      rec.Language,
		GeneratedAt:   &gen,
	})
}

func toAPINote(n *entity.Note) notesapi.Note {
	return notesapi.Note{
		Id:            n.ID,
		ShowId:        n.ShowID,
		Scope:         notesapi.NoteScope(n.Scope),
		SeasonNumber:  n.SeasonNumber,
		EpisodeNumber: n.EpisodeNumber,
		Body:          n.Body,
		CreatedAt:     n.CreatedAt.Format(time.RFC3339),
	}
}
