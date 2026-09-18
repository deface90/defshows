package workers_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/workers"
)

type fakeCatalog struct {
	candidates []entity.Show
	imported   []int64
	failOn     map[int64]bool
	listErr    error
}

func (f *fakeCatalog) ShowsForRefresh(context.Context, time.Time, int) ([]entity.Show, error) {
	return f.candidates, f.listErr
}

func (f *fakeCatalog) ImportShow(_ context.Context, tmdbID int64) (*entity.Show, error) {
	if f.failOn[tmdbID] {
		return nil, errors.New("provider boom")
	}
	f.imported = append(f.imported, tmdbID)
	return &entity.Show{TMDBID: tmdbID}, nil
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCatalogSyncer_RunOnce(t *testing.T) {
	fc := &fakeCatalog{
		candidates: []entity.Show{{TMDBID: 1}, {TMDBID: 2}, {TMDBID: 3}},
		failOn:     map[int64]bool{2: true},
	}
	s := workers.NewCatalogSyncer(fc, quietLogger(), time.Hour, 0, 100)

	refreshed, failed, err := s.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if refreshed != 2 || failed != 1 {
		t.Fatalf("want refreshed=2 failed=1, got %d/%d", refreshed, failed)
	}
	if len(fc.imported) != 2 {
		t.Fatalf("want 2 imported, got %v", fc.imported)
	}
}

func TestCatalogSyncer_RunOnce_ListError(t *testing.T) {
	fc := &fakeCatalog{listErr: errors.New("db down")}
	s := workers.NewCatalogSyncer(fc, quietLogger(), time.Hour, 0, 100)
	if _, _, err := s.RunOnce(context.Background()); err == nil {
		t.Fatal("expected error when listing candidates fails")
	}
}

func TestCatalogSyncer_RunOnce_ContextCancelled(t *testing.T) {
	fc := &fakeCatalog{candidates: []entity.Show{{TMDBID: 1}, {TMDBID: 2}}}
	s := workers.NewCatalogSyncer(fc, quietLogger(), time.Hour, 0, 100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := s.RunOnce(ctx); err == nil {
		t.Fatal("expected context error")
	}
}
