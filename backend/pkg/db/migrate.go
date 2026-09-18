package db

import (
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// RunMigrations applies all pending goose migrations from fsys/dir to the
// database behind gdb.
func RunMigrations(gdb *gorm.DB, fsys fs.FS, dir string) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return fmt.Errorf("db: sql handle: %w", err)
	}
	goose.SetBaseFS(fsys)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db: goose dialect: %w", err)
	}
	if err := goose.Up(sqlDB, dir); err != nil {
		return fmt.Errorf("db: migrate up: %w", err)
	}
	return nil
}

// MigrateReset rolls all migrations back to zero. Primarily used in tests.
func MigrateReset(gdb *gorm.DB, fsys fs.FS, dir string) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return fmt.Errorf("db: sql handle: %w", err)
	}
	goose.SetBaseFS(fsys)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db: goose dialect: %w", err)
	}
	if err := goose.DownTo(sqlDB, dir, 0); err != nil {
		return fmt.Errorf("db: migrate reset: %w", err)
	}
	return nil
}

// MigrateDown rolls back the most recent migration. Primarily used in tests.
func MigrateDown(gdb *gorm.DB, fsys fs.FS, dir string) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return fmt.Errorf("db: sql handle: %w", err)
	}
	goose.SetBaseFS(fsys)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db: goose dialect: %w", err)
	}
	if err := goose.Down(sqlDB, dir); err != nil {
		return fmt.Errorf("db: migrate down: %w", err)
	}
	return nil
}
