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
	finales  []repository.EventCandidate
	seen     map[string]bool
}

func (f *fakeScannerRepo) ReleasedEpisodeCandidates(context.Context, time.Time) ([]repository.EventCandidate, error) {
	return f.released, nil
}

func (f *fakeScannerRepo) SeasonFinaleCandidates(context.Context, time.Time) ([]repository.EventCandidate, error) {
	return f.finales, nil
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
	apnsToken := "apns-device-token"
	fcmToken := "fcm-device-token"
	repo := &fakeScannerRepo{released: []repository.EventCandidate{
		{UserID: 1, ShowID: 10, EpisodeID: 100, ShowTitle: "S", SeasonNumber: 1, EpisodeNumber: 1, TelegramChatID: &chat, APNsToken: &apnsToken, FCMToken: &fcmToken},
	}}
	s := workers.NewNotifyScanner(repo, quietLogger(), 7*24*time.Hour)

	created, err := s.RunOnce(context.Background())
	if err != nil || created != 3 {
		t.Fatalf("want 3 (telegram+apns+fcm), got created=%d err=%v", created, err)
	}
}

func TestNotifyScanner_SeasonFinale(t *testing.T) {
	chat := int64(1)
	repo := &fakeScannerRepo{finales: []repository.EventCandidate{
		{UserID: 1, ShowID: 10, EpisodeID: 108, ShowTitle: "S", SeasonNumber: 1, EpisodeNumber: 8, TelegramChatID: &chat},
	}}
	s := workers.NewNotifyScanner(repo, quietLogger(), 7*24*time.Hour)

	created, err := s.RunOnce(context.Background())
	if err != nil || created != 1 {
		t.Fatalf("first run: created=%d err=%v", created, err)
	}
	if !repo.seen["finale:1:108:telegram"] {
		t.Fatalf("expected a season-finale notification with the finale dedupe tag, seen=%v", repo.seen)
	}
	// A re-scan must not duplicate the finale notification.
	created, err = s.RunOnce(context.Background())
	if err != nil || created != 0 {
		t.Fatalf("second run should dedupe: created=%d err=%v", created, err)
	}
}
