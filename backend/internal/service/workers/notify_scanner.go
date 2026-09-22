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
	ReleasedEpisodeCandidates(ctx context.Context, since time.Time) ([]repository.EventCandidate, error)
	EpisodeUpcomingCandidates(ctx context.Context, now time.Time) ([]repository.EventCandidate, error)
	SeasonUpcomingCandidates(ctx context.Context, now time.Time) ([]repository.EventCandidate, error)
	CreateNotificationIfAbsent(ctx context.Context, n *entity.Notification) (bool, error)
}

// NotifyScanner creates notifications for newly-aired, unwatched episodes and
// for upcoming episodes/season premieres of watched shows.
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

// eventKind describes one notification type's candidate source and how to
// render its payload text.
type eventKind struct {
	notifyType string
	dedupeTag  string
	payload    func(c repository.EventCandidate) string
}

var eventKinds = []eventKind{
	{
		notifyType: entity.NotifyEpisodeReleased,
		dedupeTag:  "release",
		payload: func(c repository.EventCandidate) string {
			return fmt.Sprintf("Вышла новая серия: %s S%02dE%02d", c.ShowTitle, c.SeasonNumber, c.EpisodeNumber)
		},
	},
	{
		notifyType: entity.NotifyEpisodeUpcoming,
		dedupeTag:  "upcoming",
		payload: func(c repository.EventCandidate) string {
			return fmt.Sprintf("Скоро выйдет серия: %s S%02dE%02d", c.ShowTitle, c.SeasonNumber, c.EpisodeNumber)
		},
	},
	{
		notifyType: entity.NotifySeasonUpcoming,
		dedupeTag:  "season",
		payload: func(c repository.EventCandidate) string {
			return fmt.Sprintf("Скоро новый сезон: %s, сезон %d", c.ShowTitle, c.SeasonNumber)
		},
	},
}

// RunOnce scans for release/upcoming candidates and enqueues notifications
// (deduped) on every channel available for each user.
func (s *NotifyScanner) RunOnce(ctx context.Context) (created int, err error) {
	now := time.Now()
	since := now.Add(-s.lookback)

	candidatesByKind := map[string][]repository.EventCandidate{}
	candidatesByKind[entity.NotifyEpisodeReleased], err = s.repo.ReleasedEpisodeCandidates(ctx, since)
	if err != nil {
		return 0, err
	}
	candidatesByKind[entity.NotifyEpisodeUpcoming], err = s.repo.EpisodeUpcomingCandidates(ctx, now)
	if err != nil {
		return 0, err
	}
	candidatesByKind[entity.NotifySeasonUpcoming], err = s.repo.SeasonUpcomingCandidates(ctx, now)
	if err != nil {
		return 0, err
	}

	for _, kind := range eventKinds {
		for _, c := range candidatesByKind[kind.notifyType] {
			if ctx.Err() != nil {
				return created, ctx.Err()
			}
			created += s.enqueue(ctx, kind, c)
		}
	}
	return created, nil
}

// enqueue creates one notification per channel available for the candidate
// (telegram and/or apns), returning how many rows were newly created.
func (s *NotifyScanner) enqueue(ctx context.Context, kind eventKind, c repository.EventCandidate) int {
	created := 0
	targets := []string{}
	if c.TelegramChatID != nil {
		targets = append(targets, "telegram")
	}
	if c.APNsToken != nil {
		targets = append(targets, "apns")
	}
	showID, episodeID := c.ShowID, c.EpisodeID
	for _, channel := range targets {
		n := &entity.Notification{
			UserID:       c.UserID,
			Type:         kind.notifyType,
			ShowID:       &showID,
			EpisodeID:    &episodeID,
			Channel:      channel,
			Status:       entity.NotifyPending,
			ScheduledFor: time.Now(),
			DedupeKey:    fmt.Sprintf("%s:%d:%d:%s", kind.dedupeTag, c.UserID, c.EpisodeID, channel),
			Payload:      kind.payload(c),
		}
		ok, err := s.repo.CreateNotificationIfAbsent(ctx, n)
		if err != nil {
			s.logger.Warn("create notification failed", "user_id", c.UserID, "episode_id", c.EpisodeID, "channel", channel, "err", err)
			continue
		}
		if ok {
			created++
		}
	}
	return created
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
