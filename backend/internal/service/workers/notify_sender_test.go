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
}

func (f *fakeSenderRepo) PendingTelegram(context.Context, int) ([]repository.PendingNotification, error) {
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

type fakeChannel struct {
	failChatID int64
}

func (f fakeChannel) Send(_ context.Context, msg notify.Message) error {
	if msg.ChatID == f.failChatID {
		return errors.New("boom")
	}
	return nil
}

func TestNotifySender_RunOnce(t *testing.T) {
	repo := &fakeSenderRepo{pending: []repository.PendingNotification{
		{ID: 1, ChatID: 111, Payload: "ok"},
		{ID: 2, ChatID: 222, Payload: "boom"},
	}}
	s := workers.NewNotifySender(repo, fakeChannel{failChatID: 222}, quietLogger(), 100)

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
