package db_test

import (
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/testutil"
	"github.com/deface90/defshows/backend/pkg/config"
	"github.com/deface90/defshows/backend/pkg/db"
)

func TestConnect_PingsRealPostgres(t *testing.T) {
	dsn := testutil.PostgresDSN(t)

	gdb, err := db.Connect(config.DB{
		DSN:             dsn,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestConnect_UnreachableDSN(t *testing.T) {
	_, err := db.Connect(config.DB{
		DSN: "postgres://bad:bad@127.0.0.1:1/none?sslmode=disable&connect_timeout=1",
	})
	if err == nil {
		t.Fatal("expected error for unreachable DSN, got nil")
	}
}
