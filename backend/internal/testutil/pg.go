// Package testutil provides shared test helpers, notably an ephemeral Postgres
// backed by testcontainers. One container is started per test binary (package)
// and reaped by Ryuk at the end of the run.
package testutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"

	"github.com/deface90/defshows/backend/migrations"
	"github.com/deface90/defshows/backend/pkg/config"
	pkgdb "github.com/deface90/defshows/backend/pkg/db"
)

var (
	once      sync.Once
	sharedDSN string
	initErr   error

	migrateOnce sync.Once
	migrateErr  error
)

// PostgresDSN starts (once per test binary) a throwaway Postgres container and
// returns its DSN. If Docker is unavailable the calling test is skipped rather
// than failed, so pure-logic suites still run in constrained environments.
func PostgresDSN(t testing.TB) string {
	t.Helper()
	once.Do(func() {
		ctx := context.Background()
		container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
			tcpostgres.WithDatabase("defshows_test"),
			tcpostgres.WithUsername("test"),
			tcpostgres.WithPassword("test"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second),
			),
		)
		if err != nil {
			initErr = err
			return
		}
		sharedDSN, initErr = container.ConnectionString(ctx, "sslmode=disable")
	})
	if initErr != nil {
		t.Skipf("skipping: cannot start postgres container: %v", initErr)
	}
	return sharedDSN
}

// PostgresDB returns a gorm connection to the shared test Postgres.
func PostgresDB(t testing.TB) *gorm.DB {
	t.Helper()
	gdb, err := pkgdb.Connect(config.DB{
		DSN:             PostgresDSN(t),
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	return gdb
}

// MigratedPostgresDB returns a gorm connection to the shared test Postgres with
// all migrations applied (once per test binary).
func MigratedPostgresDB(t testing.TB) *gorm.DB {
	t.Helper()
	gdb := PostgresDB(t)
	migrateOnce.Do(func() {
		migrateErr = pkgdb.RunMigrations(gdb, migrations.FS, migrations.Dir)
	})
	if migrateErr != nil {
		t.Fatalf("migrate test db: %v", migrateErr)
	}
	return gdb
}

// Truncate empties the given tables (RESTART IDENTITY, CASCADE) for test isolation.
func Truncate(t testing.TB, gdb *gorm.DB, tables ...string) {
	t.Helper()
	for _, tbl := range tables {
		if err := gdb.Exec("TRUNCATE " + tbl + " RESTART IDENTITY CASCADE").Error; err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
}
