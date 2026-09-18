package crud_test

import (
	"context"
	"errors"
	"testing"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/testutil"
	"github.com/deface90/defshows/backend/pkg/crud"
)

func TestRepository_CRUD(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "dubbing_studios")
	repo := crud.NewRepository[entity.DubbingStudio](gdb)
	ctx := context.Background()

	// Create.
	ds := &entity.DubbingStudio{Name: "LostFilm", SiteURL: "https://lostfilm.tv", Active: true}
	if err := repo.Create(ctx, ds); err != nil {
		t.Fatalf("create: %v", err)
	}
	if ds.ID == 0 {
		t.Fatal("expected generated id")
	}

	// Get.
	got, err := repo.Get(ctx, ds.ID)
	if err != nil || got.Name != "LostFilm" {
		t.Fatalf("get: %v (%+v)", err, got)
	}

	// Save (update).
	got.Active = false
	if err := repo.Save(ctx, got); err != nil {
		t.Fatalf("save: %v", err)
	}
	reread, _ := repo.Get(ctx, ds.ID)
	if reread.Active {
		t.Fatal("update not applied")
	}

	// List.
	_ = repo.Create(ctx, &entity.DubbingStudio{Name: "HDrezka", Active: true})
	list, err := repo.List(ctx, 10, 0)
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %v (n=%d)", err, len(list))
	}

	// Delete.
	if err := repo.Delete(ctx, ds.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, ds.ID); !errors.Is(err, crud.ErrNotFound) {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
	if err := repo.Delete(ctx, 99999); !errors.Is(err, crud.ErrNotFound) {
		t.Fatalf("want ErrNotFound deleting missing, got %v", err)
	}
}
