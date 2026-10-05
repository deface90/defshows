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
	SeasonFinaleCandidates(ctx context.Context, since time.Time) ([]repository.EventCandidate, error)
	EpisodeUpcomingCandidates(ctx context.Context, now time.Time) ([]repository.EventCandidate, error)
	SeasonUpcomingCandidates(ctx context.Context, now time.Time) ([]repository.EventCandidate, error)
	EnqueueNotification(ctx context.Context, n *entity.Notification, channels []string) (bool, error)
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
	{
		notifyType: entity.NotifySeasonFinale,
		dedupeTag:  "finale",
		payload: func(c repository.EventCandidate) string {
			return fmt.Sprintf("Сезон завершён: %s, сезон %d", c.ShowTitle, c.SeasonNumber)
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
	candidatesByKind[entity.NotifySeasonFinale], err = s.repo.SeasonFinaleCandidates(ctx, since)
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

// enqueue creates one notification (feed event) for the candidate, delivered to every
// channel available for the user (telegram and/or apns and/or fcm). Returns 1 if the event
// was newly created, 0 on a dedupe hit or if the user has no delivery channel.
func (s *NotifyScanner) enqueue(ctx context.Context, kind eventKind, c repository.EventCandidate) int {
	channels := candidateChannels(c)
	if len(channels) == 0 {
		return 0
	}
	showID, episodeID := c.ShowID, c.EpisodeID
	n := &entity.Notification{
		UserID:    c.UserID,
		Type:      kind.notifyType,
		ShowID:    &showID,
		EpisodeID: &episodeID,
		DedupeKey: fmt.Sprintf("%s:%d:%d", kind.dedupeTag, c.UserID, c.EpisodeID),
		Payload:   kind.payload(c),
	}
	created, err := s.repo.EnqueueNotification(ctx, n, channels)
	if err != nil {
		s.logger.Warn("enqueue notification failed", "user_id", c.UserID, "episode_id", c.EpisodeID, "err", err)
		return 0
	}
	if created {
		return 1
	}
	return 0
}

// candidateChannels lists the delivery channels currently linked for the candidate's user.
func candidateChannels(c repository.EventCandidate) []string {
	channels := []string{}
	if c.TelegramChatID != nil {
		channels = append(channels, "telegram")
	}
	if c.APNsToken != nil {
		channels = append(channels, "apns")
	}
	if c.FCMToken != nil {
		channels = append(channels, "fcm")
	}
	return channels
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
