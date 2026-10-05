package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
)

// NotificationBackfillRepo is the storage dependency of NotificationBackfill. It is the
// notification repository in production; the candidate queries are the exact ones the
// scanner uses, so the seeded dedupe keys match what the scanner would otherwise emit.
type NotificationBackfillRepo interface {
	ReleasedEpisodeCandidates(ctx context.Context, since time.Time) ([]repository.EventCandidate, error)
	SeasonFinaleCandidates(ctx context.Context, since time.Time) ([]repository.EventCandidate, error)
	EnqueueNotification(ctx context.Context, n *entity.Notification, channels []string) (bool, error)
}

// NotificationBackfill seeds "already handled" markers for episodes that have already
// aired, so enabling a previously-broken delivery channel (e.g. APNs) does not fire a
// burst of catch-up pushes for a user's existing backlog on first run.
//
// Each marker is a normal feed notification (visible in the in-app feed, pre-marked read
// so it doesn't inflate the unread badge) but carries NO deliveries — so nothing is ever
// pushed. Because it occupies the scanner's dedupe_key (`release:<user>:<episode>` /
// `finale:<user>:<episode>`), the scanner's later EnqueueNotification is a no-op for that
// episode. It is idempotent (dedupe) and safe to re-run; genuinely new episodes (aired
// after the backfill) are unaffected and notify normally.
type NotificationBackfill struct {
	repo NotificationBackfillRepo
}

// NewNotificationBackfill creates a NotificationBackfill.
func NewNotificationBackfill(repo NotificationBackfillRepo) *NotificationBackfill {
	return &NotificationBackfill{repo: repo}
}

// Run seeds markers for episodes aired within the window (the same lookback the scanner
// uses) and returns how many were newly created. Zero means nothing needed seeding.
func (b *NotificationBackfill) Run(ctx context.Context, window time.Duration) (int, error) {
	if window <= 0 {
		window = 7 * 24 * time.Hour
	}
	since := time.Now().Add(-window)
	read := time.Now()
	seeded := 0

	seed := func(c repository.EventCandidate, notifyType, dedupe, payload string) (bool, error) {
		showID, episodeID := c.ShowID, c.EpisodeID
		n := &entity.Notification{
			UserID:    c.UserID,
			Type:      notifyType,
			ShowID:    &showID,
			EpisodeID: &episodeID,
			DedupeKey: dedupe,
			Payload:   payload,
			ReadAt:    &read,
		}
		// No channels → no deliveries → never pushed; the row only reserves the dedupe key.
		return b.repo.EnqueueNotification(ctx, n, nil)
	}

	released, err := b.repo.ReleasedEpisodeCandidates(ctx, since)
	if err != nil {
		return seeded, err
	}
	for _, c := range released {
		created, err := seed(c, entity.NotifyEpisodeReleased,
			fmt.Sprintf("release:%d:%d", c.UserID, c.EpisodeID),
			fmt.Sprintf("Вышла новая серия: %s S%02dE%02d", c.ShowTitle, c.SeasonNumber, c.EpisodeNumber))
		if err != nil {
			return seeded, err
		}
		if created {
			seeded++
		}
	}

	finales, err := b.repo.SeasonFinaleCandidates(ctx, since)
	if err != nil {
		return seeded, err
	}
	for _, c := range finales {
		created, err := seed(c, entity.NotifySeasonFinale,
			fmt.Sprintf("finale:%d:%d", c.UserID, c.EpisodeID),
			fmt.Sprintf("Сезон завершён: %s, сезон %d", c.ShowTitle, c.SeasonNumber))
		if err != nil {
			return seeded, err
		}
		if created {
			seeded++
		}
	}

	return seeded, nil
}
