package workers_test

import (
	"context"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/workers"
)

type fakeScannerRepo struct {
	candidates []repository.ReleaseCandidate
	seen       map[string]bool
}

func (f *fakeScannerRepo) ReleasedEpisodeCandidates(context.Context, time.Time) ([]repository.ReleaseCandidate, error) {
	return f.candidates, nil
}

func (f *fakeScannerRepo) CreateNotificationIfAbsent(_ context.Context, n *entity.Notification) (bool, error) {
	if f.seen == nil {
		f.seen = map[string]bool{}
	}
	if f.seen[n.DedupeKey] {
		return false, nil
	}
	f.seen[n.DedupeKey] = true
	return true, nil
}

func TestNotifyScanner_DedupesAcrossRuns(t *testing.T) {
	repo := &fakeScannerRepo{candidates: []repository.ReleaseCandidate{
		{UserID: 1, ShowID: 10, EpisodeID: 100, ShowTitle: "S", SeasonNumber: 1, EpisodeNumber: 1},
		{UserID: 1, ShowID: 10, EpisodeID: 101, ShowTitle: "S", SeasonNumber: 1, EpisodeNumber: 2},
	}}
	s := workers.NewNotifyScanner(repo, quietLogger(), 7*24*time.Hour)

	created, err := s.RunOnce(context.Background())
	if err != nil || created != 2 {
		t.Fatalf("first run: created=%d err=%v", created, err)
	}
	created, err = s.RunOnce(context.Background())
	if err != nil || created != 0 {
		t.Fatalf("second run should dedupe: created=%d err=%v", created, err)
	}
}
