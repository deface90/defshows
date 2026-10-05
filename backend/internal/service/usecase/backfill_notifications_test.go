package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
)

type fakeBackfillRepo struct {
	released []repository.EventCandidate
	finales  []repository.EventCandidate
	enqueued []entity.Notification
	channels map[string][]string
}

func (f *fakeBackfillRepo) ReleasedEpisodeCandidates(_ context.Context, _ time.Time) ([]repository.EventCandidate, error) {
	return f.released, nil
}

func (f *fakeBackfillRepo) SeasonFinaleCandidates(_ context.Context, _ time.Time) ([]repository.EventCandidate, error) {
	return f.finales, nil
}

func (f *fakeBackfillRepo) EnqueueNotification(_ context.Context, n *entity.Notification, channels []string) (bool, error) {
	for i := range f.enqueued {
		if f.enqueued[i].DedupeKey == n.DedupeKey {
			return false, nil
		}
	}
	f.enqueued = append(f.enqueued, *n)
	if f.channels == nil {
		f.channels = map[string][]string{}
	}
	f.channels[n.DedupeKey] = channels
	return true, nil
}

func TestNotificationBackfill_SeedsMarkersWithoutDeliveries(t *testing.T) {
	ctx := context.Background()
	repo := &fakeBackfillRepo{
		released: []repository.EventCandidate{
			{UserID: 1, ShowID: 10, EpisodeID: 100, ShowTitle: "S", SeasonNumber: 1, EpisodeNumber: 3},
		},
		finales: []repository.EventCandidate{
			{UserID: 1, ShowID: 10, EpisodeID: 108, ShowTitle: "S", SeasonNumber: 1, EpisodeNumber: 8},
		},
	}

	seeded, err := usecase.NewNotificationBackfill(repo).Run(ctx, 7*24*time.Hour)
	if err != nil || seeded != 2 {
		t.Fatalf("run: seeded=%d err=%v", seeded, err)
	}

	// Markers use the scanner's dedupe keys so the scanner later no-ops on these episodes.
	keys := map[string]bool{}
	for _, n := range repo.enqueued {
		keys[n.DedupeKey] = true
		// Crucially: NO delivery channels → nothing is ever pushed.
		if ch := repo.channels[n.DedupeKey]; len(ch) != 0 {
			t.Fatalf("marker %q must have no deliveries, got %v", n.DedupeKey, ch)
		}
		// Pre-marked read so it doesn't inflate the unread badge.
		if n.ReadAt == nil {
			t.Fatalf("marker %q should be pre-marked read", n.DedupeKey)
		}
	}
	if !keys["release:1:100"] || !keys["finale:1:108"] {
		t.Fatalf("unexpected dedupe keys: %v", keys)
	}

	// Idempotent: a re-run seeds nothing.
	seeded, err = usecase.NewNotificationBackfill(repo).Run(ctx, 7*24*time.Hour)
	if err != nil || seeded != 0 {
		t.Fatalf("re-run should be a no-op: seeded=%d err=%v", seeded, err)
	}
}
