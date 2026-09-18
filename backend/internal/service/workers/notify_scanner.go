package workers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
)

// ScannerRepo is the storage dependency of NotifyScanner.
type ScannerRepo interface {
	ReleasedEpisodeCandidates(ctx context.Context, since time.Time) ([]repository.ReleaseCandidate, error)
	CreateNotificationIfAbsent(ctx context.Context, n *entity.Notification) (bool, error)
}

// NotifyScanner creates notifications for newly-aired, unwatched episodes.
type NotifyScanner struct {
	repo     ScannerRepo
	logger   *slog.Logger
	lookback time.Duration
}

// NewNotifyScanner creates a NotifyScanner. lookback bounds how far back aired
// episodes are considered (avoids backfilling entire history).
func NewNotifyScanner(repo ScannerRepo, logger *slog.Logger, lookback time.Duration) *NotifyScanner {
	if lookback <= 0 {
		lookback = 7 * 24 * time.Hour
	}
	return &NotifyScanner{repo: repo, logger: logger, lookback: lookback}
}

// RunOnce scans for release candidates and enqueues notifications (deduped).
func (s *NotifyScanner) RunOnce(ctx context.Context) (created int, err error) {
	since := time.Now().Add(-s.lookback)
	candidates, err := s.repo.ReleasedEpisodeCandidates(ctx, since)
	if err != nil {
		return 0, err
	}
	for _, c := range candidates {
		if ctx.Err() != nil {
			return created, ctx.Err()
		}
		showID, episodeID := c.ShowID, c.EpisodeID
		n := &entity.Notification{
			UserID:       c.UserID,
			Type:         entity.NotifyEpisodeReleased,
			ShowID:       &showID,
			EpisodeID:    &episodeID,
			Channel:      "telegram",
			Status:       entity.NotifyPending,
			ScheduledFor: time.Now(),
			DedupeKey:    fmt.Sprintf("release:%d:%d:telegram", c.UserID, c.EpisodeID),
			Payload:      fmt.Sprintf("Вышла новая серия: %s S%02dE%02d", c.ShowTitle, c.SeasonNumber, c.EpisodeNumber),
		}
		ok, err := s.repo.CreateNotificationIfAbsent(ctx, n)
		if err != nil {
			s.logger.Warn("create notification failed", "user_id", c.UserID, "episode_id", c.EpisodeID, "err", err)
			continue
		}
		if ok {
			created++
		}
	}
	return created, nil
}

// Run scans on an interval until the context is cancelled.
func (s *NotifyScanner) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if created, err := s.RunOnce(ctx); err != nil && ctx.Err() == nil {
			s.logger.Error("notify scan failed", "err", err)
		} else {
			s.logger.Info("notify scan complete", "created", created)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
