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
	released []repository.EventCandidate
	seen     map[string]bool
}

func (f *fakeScannerRepo) ReleasedEpisodeCandidates(context.Context, time.Time) ([]repository.EventCandidate, error) {
	return f.released, nil
}

func (f *fakeScannerRepo) EpisodeUpcomingCandidates(context.Context, time.Time) ([]repository.EventCandidate, error) {
	return nil, nil
}

func (f *fakeScannerRepo) SeasonUpcomingCandidates(context.Context, time.Time) ([]repository.EventCandidate, error) {
	return nil, nil
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
	chat := int64(1)
	repo := &fakeScannerRepo{released: []repository.EventCandidate{
		{UserID: 1, ShowID: 10, EpisodeID: 100, ShowTitle: "S", SeasonNumber: 1, EpisodeNumber: 1, TelegramChatID: &chat},
		{UserID: 1, ShowID: 10, EpisodeID: 101, ShowTitle: "S", SeasonNumber: 1, EpisodeNumber: 2, TelegramChatID: &chat},
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

func TestNotifyScanner_FansOutAcrossChannels(t *testing.T) {
	chat := int64(1)
	token := "device-token"
	repo := &fakeScannerRepo{released: []repository.EventCandidate{
		{UserID: 1, ShowID: 10, EpisodeID: 100, ShowTitle: "S", SeasonNumber: 1, EpisodeNumber: 1, TelegramChatID: &chat, APNsToken: &token},
	}}
	s := workers.NewNotifyScanner(repo, quietLogger(), 7*24*time.Hour)

	created, err := s.RunOnce(context.Background())
	if err != nil || created != 2 {
		t.Fatalf("want 2 (telegram+apns), got created=%d err=%v", created, err)
	}
}
