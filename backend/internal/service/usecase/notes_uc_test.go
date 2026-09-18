package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/internal/testutil"
)

func TestNotesUsecase_OwnershipAndRecap(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows")
	ctx := context.Background()

	userRepo := repository.NewUserRepository(gdb)
	a := &entity.User{Email: strp("a@ex.com"), Role: entity.RoleUser, Timezone: "UTC"}
	b := &entity.User{Email: strp("b@ex.com"), Role: entity.RoleUser, Timezone: "UTC"}
	_ = userRepo.CreateUser(ctx, a)
	_ = userRepo.CreateUser(ctx, b)

	catalogRepo := repository.NewCatalogRepository(gdb)
	show := &entity.Show{TMDBID: 1, Title: "S"}
	_ = catalogRepo.UpsertShow(ctx, show)

	notesRepo := repository.NewNotesRepository(gdb)
	uc := usecase.NewNotesUsecase(notesRepo)

	note, err := uc.CreateNote(ctx, a.ID, show.ID, "show", nil, nil, "my note")
	if err != nil {
		t.Fatalf("create note: %v", err)
	}

	// Owner can update.
	updated, err := uc.UpdateNote(ctx, a.ID, note.ID, "edited")
	if err != nil || updated.Body != "edited" {
		t.Fatalf("update: %v (%+v)", err, updated)
	}

	// Non-owner cannot update or delete.
	if _, err := uc.UpdateNote(ctx, b.ID, note.ID, "hacked"); !errors.Is(err, usecase.ErrNoteNotFound) {
		t.Fatalf("want ErrNoteNotFound for non-owner update, got %v", err)
	}
	if err := uc.DeleteNote(ctx, b.ID, note.ID); !errors.Is(err, usecase.ErrNoteNotFound) {
		t.Fatalf("want ErrNoteNotFound for non-owner delete, got %v", err)
	}

	// Owner list sees the note; other user sees none.
	if notes, _ := uc.ListNotes(ctx, a.ID, show.ID); len(notes) != 1 {
		t.Fatalf("owner list: want 1, got %d", len(notes))
	}
	if notes, _ := uc.ListNotes(ctx, b.ID, show.ID); len(notes) != 0 {
		t.Fatalf("other list: want 0, got %d", len(notes))
	}

	// Recap not available yet.
	if _, err := uc.GetRecap(ctx, show.ID, "show", nil, nil, "ru"); !errors.Is(err, usecase.ErrRecapNotAvailable) {
		t.Fatalf("want ErrRecapNotAvailable, got %v", err)
	}
	// Seed a recap and read it; re-upsert updates (dedup).
	rec := &entity.Recap{ShowID: show.ID, Scope: entity.ScopeShow, Body: "recap v1", Language: "ru", GeneratedAt: time.Now()}
	if err := notesRepo.UpsertRecap(ctx, rec); err != nil {
		t.Fatalf("seed recap: %v", err)
	}
	rec.Body = "recap v2"
	if err := notesRepo.UpsertRecap(ctx, rec); err != nil {
		t.Fatalf("re-upsert recap: %v", err)
	}
	got, err := uc.GetRecap(ctx, show.ID, "show", nil, nil, "ru")
	if err != nil || got.Body != "recap v2" {
		t.Fatalf("get recap: %v (%+v)", err, got)
	}

	// Owner delete works.
	if err := uc.DeleteNote(ctx, a.ID, note.ID); err != nil {
		t.Fatalf("owner delete: %v", err)
	}
}

func strp(s string) *string { return &s }
