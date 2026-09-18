package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	authapi "github.com/deface90/defshows/backend/pkg/server/auth"
	notesapi "github.com/deface90/defshows/backend/pkg/server/notes"
	trackingapi "github.com/deface90/defshows/backend/pkg/server/tracking"
)

func TestNotesHandler_CRUDAndRecap(t *testing.T) {
	e := newWebServer(t)

	// Register + add a show (so a valid show_id exists).
	rec := doJSON(t, e, http.MethodPost, "/auth/register", "", map[string]string{
		"email": "notes@example.com", "password": "pw12345",
	})
	var reg authapi.AuthResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)
	token := reg.Tokens.AccessToken

	rec = doJSON(t, e, http.MethodPost, "/me/shows", token, map[string]int64{"tmdb_id": 1399})
	var us trackingapi.UserShow
	_ = json.Unmarshal(rec.Body.Bytes(), &us)
	showID := us.ShowId

	// Create note.
	rec = doJSON(t, e, http.MethodPost, "/me/notes", token, map[string]any{
		"show_id": showID, "scope": "show", "body": "note body",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create note: %d (%s)", rec.Code, rec.Body.String())
	}
	var note notesapi.Note
	_ = json.Unmarshal(rec.Body.Bytes(), &note)

	// List.
	rec = doJSON(t, e, http.MethodGet, "/me/notes?show_id="+strconv.FormatInt(showID, 10), token, nil)
	var list notesapi.NoteList
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Notes) != 1 {
		t.Fatalf("want 1 note, got %d", len(list.Notes))
	}

	// Update.
	rec = doJSON(t, e, http.MethodPatch, "/me/notes/"+strconv.FormatInt(note.Id, 10), token,
		map[string]string{"body": "edited"})
	if rec.Code != http.StatusOK {
		t.Fatalf("update note: %d", rec.Code)
	}

	// Recap not available → 404.
	rec = doJSON(t, e, http.MethodGet, "/shows/"+strconv.FormatInt(showID, 10)+"/recap?scope=show", token, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("recap: want 404, got %d", rec.Code)
	}

	// Delete.
	rec = doJSON(t, e, http.MethodDelete, "/me/notes/"+strconv.FormatInt(note.Id, 10), token, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete note: %d", rec.Code)
	}
	// Delete again → 404.
	rec = doJSON(t, e, http.MethodDelete, "/me/notes/"+strconv.FormatInt(note.Id, 10), token, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete again: want 404, got %d", rec.Code)
	}
}
