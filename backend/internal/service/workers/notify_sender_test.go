package workers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/workers"
	"github.com/deface90/defshows/backend/pkg/notify"
)

type fakeSenderRepo struct {
	pending []repository.PendingNotification
	sent    []int64
	failed  []int64
	cleared []string
}

func (f *fakeSenderRepo) Pending(context.Context, string, int) ([]repository.PendingNotification, error) {
	return f.pending, nil
}
func (f *fakeSenderRepo) MarkSent(_ context.Context, id int64) error {
	f.sent = append(f.sent, id)
	return nil
}
func (f *fakeSenderRepo) MarkFailed(_ context.Context, id int64) error {
	f.failed = append(f.failed, id)
	return nil
}
func (f *fakeSenderRepo) ClearTarget(_ context.Context, _, target string) error {
	f.cleared = append(f.cleared, target)
	return nil
}

type fakeChannel struct {
	failTarget  string
	unregTarget string
}

func (f fakeChannel) Send(_ context.Context, msg notify.Message) error {
	if msg.Target == f.unregTarget {
		return &notify.UnregisteredTargetError{Target: msg.Target}
	}
	if msg.Target == f.failTarget {
		return errors.New("boom")
	}
	return nil
}

func TestNotifySender_RunOnce(t *testing.T) {
	repo := &fakeSenderRepo{pending: []repository.PendingNotification{
		{ID: 1, Target: "111", Body: "ok"},
		{ID: 2, Target: "222", Body: "boom"},
	}}
	s := workers.NewNotifySender(repo, "telegram", fakeChannel{failTarget: "222"}, quietLogger(), 100)

	sent, failed, err := s.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if sent != 1 || failed != 1 {
		t.Fatalf("want sent=1 failed=1, got %d/%d", sent, failed)
	}
	if len(repo.sent) != 1 || repo.sent[0] != 1 {
		t.Fatalf("marked sent: %v", repo.sent)
	}
	if len(repo.failed) != 1 || repo.failed[0] != 2 {
		t.Fatalf("marked failed: %v", repo.failed)
	}
}

func TestNotifySender_PurgesUnregisteredTarget(t *testing.T) {
	repo := &fakeSenderRepo{pending: []repository.PendingNotification{
		{ID: 1, Target: "live-token", Body: "ok"},
		{ID: 2, Target: "dead-token", Body: "gone"},
	}}
	s := workers.NewNotifySender(repo, "apns", fakeChannel{unregTarget: "dead-token"}, quietLogger(), 100)

	sent, failed, err := s.RunOnce(context.Background())
	if err != nil || sent != 1 || failed != 1 {
		t.Fatalf("run once: sent=%d failed=%d err=%v", sent, failed, err)
	}
	// The dead token is purged (so it won't be retried) and its delivery marked failed.
	if len(repo.cleared) != 1 || repo.cleared[0] != "dead-token" {
		t.Fatalf("cleared: %v", repo.cleared)
	}
	if len(repo.failed) != 1 || repo.failed[0] != 2 {
		t.Fatalf("marked failed: %v", repo.failed)
	}
	// The live token is untouched.
	if len(repo.sent) != 1 || repo.sent[0] != 1 {
		t.Fatalf("marked sent: %v", repo.sent)
	}
}
