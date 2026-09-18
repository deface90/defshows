// Package workers holds background jobs that run over the shared database.
package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
)

// CatalogSyncPort is the catalog dependency of CatalogSyncer.
type CatalogSyncPort interface {
	ShowsForRefresh(ctx context.Context, staleBefore time.Time, limit int) ([]entity.Show, error)
	ImportShow(ctx context.Context, tmdbID int64) (*entity.Show, error)
}

// RatingsRefresher optionally refreshes ratings for a show after it is synced.
type RatingsRefresher interface {
	RefreshForShow(ctx context.Context, show *entity.Show) error
}

// CatalogSyncer periodically refreshes "live" shows from the provider.
type CatalogSyncer struct {
	catalog  CatalogSyncPort
	ratings  RatingsRefresher
	logger   *slog.Logger
	staleAge time.Duration
	batch    int
	throttle time.Duration
}

// WithRatings attaches an optional ratings refresher run after each show import.
func (s *CatalogSyncer) WithRatings(r RatingsRefresher) *CatalogSyncer {
	s.ratings = r
	return s
}

// NewCatalogSyncer creates a CatalogSyncer. staleAge controls how old a show's
// last sync must be to be refreshed; throttle spaces provider calls.
func NewCatalogSyncer(catalog CatalogSyncPort, logger *slog.Logger, staleAge, throttle time.Duration, batch int) *CatalogSyncer {
	if batch <= 0 {
		batch = 100
	}
	return &CatalogSyncer{catalog: catalog, logger: logger, staleAge: staleAge, batch: batch, throttle: throttle}
}

// RunOnce refreshes one batch of candidate shows. Provider errors on individual
// shows are logged and skipped; they do not abort the batch.
func (s *CatalogSyncer) RunOnce(ctx context.Context) (refreshed, failed int, err error) {
	staleBefore := time.Now().Add(-s.staleAge)
	shows, err := s.catalog.ShowsForRefresh(ctx, staleBefore, s.batch)
	if err != nil {
		return 0, 0, err
	}
	for _, show := range shows {
		if ctx.Err() != nil {
			return refreshed, failed, ctx.Err()
		}
		imported, err := s.catalog.ImportShow(ctx, show.TMDBID)
		if err != nil {
			failed++
			s.logger.Warn("catalog refresh failed", "tmdb_id", show.TMDBID, "err", err)
		} else {
			refreshed++
			if s.ratings != nil {
				if err := s.ratings.RefreshForShow(ctx, imported); err != nil {
					s.logger.Warn("ratings refresh failed", "tmdb_id", show.TMDBID, "err", err)
				}
			}
		}
		if s.throttle > 0 {
			select {
			case <-ctx.Done():
				return refreshed, failed, ctx.Err()
			case <-time.After(s.throttle):
			}
		}
	}
	return refreshed, failed, nil
}

// Run refreshes on an interval until the context is cancelled.
func (s *CatalogSyncer) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		refreshed, failed, err := s.RunOnce(ctx)
		if err != nil && ctx.Err() == nil {
			s.logger.Error("catalog sync run failed", "err", err)
		} else {
			s.logger.Info("catalog sync run complete", "refreshed", refreshed, "failed", failed)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
