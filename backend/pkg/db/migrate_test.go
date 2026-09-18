package db_test

import (
	"testing"

	"gorm.io/gorm"

	"github.com/deface90/defshows/backend/internal/testutil"
	"github.com/deface90/defshows/backend/migrations"
	"github.com/deface90/defshows/backend/pkg/db"
)

func TestMigrations_UpDown(t *testing.T) {
	gdb := testutil.PostgresDB(t)

	if err := db.RunMigrations(gdb, migrations.FS, migrations.Dir); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	for _, tbl := range []string{"users", "user_identities", "refresh_tokens", "shows", "seasons", "episodes", "user_shows"} {
		if !tableExists(t, gdb, tbl) {
			t.Fatalf("expected table %q to exist after up", tbl)
		}
	}

	// A single Down rolls back only the most recent migration; users (from the
	// first migration) survive.
	if err := db.MigrateDown(gdb, migrations.FS, migrations.Dir); err != nil {
		t.Fatalf("migrate down: %v", err)
	}
	if !tableExists(t, gdb, "users") {
		t.Fatal("expected users table to survive a single down")
	}

	// Full reset drops everything.
	if err := db.MigrateReset(gdb, migrations.FS, migrations.Dir); err != nil {
		t.Fatalf("migrate reset: %v", err)
	}
	if tableExists(t, gdb, "users") {
		t.Fatal("expected users table gone after reset to zero")
	}

	// Re-apply so the shared container is left fully migrated for sibling tests.
	if err := db.RunMigrations(gdb, migrations.FS, migrations.Dir); err != nil {
		t.Fatalf("re-migrate up: %v", err)
	}
	if !tableExists(t, gdb, "users") || !tableExists(t, gdb, "user_shows") {
		t.Fatal("expected tables restored after re-up")
	}
}

func tableExists(t *testing.T, gdb *gorm.DB, name string) bool {
	t.Helper()
	var exists bool
	if err := gdb.Raw(
		`SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = ?)`, name,
	).Scan(&exists).Error; err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	return exists
}
