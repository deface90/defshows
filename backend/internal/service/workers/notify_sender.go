package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/pkg/notify"
)

// SenderRepo is the storage dependency of NotifySender.
type SenderRepo interface {
	PendingTelegram(ctx context.Context, limit int) ([]repository.PendingNotification, error)
	MarkSent(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64) error
}

// NotifySender delivers pending notifications via a channel.
type NotifySender struct {
	repo    SenderRepo
	channel notify.Channel
	logger  *slog.Logger
	batch   int
}

// NewNotifySender creates a NotifySender.
func NewNotifySender(repo SenderRepo, channel notify.Channel, logger *slog.Logger, batch int) *NotifySender {
	if batch <= 0 {
		batch = 100
	}
	return &NotifySender{repo: repo, channel: channel, logger: logger, batch: batch}
}

// RunOnce sends one batch of pending notifications.
func (s *NotifySender) RunOnce(ctx context.Context) (sent, failed int, err error) {
	pending, err := s.repo.PendingTelegram(ctx, s.batch)
	if err != nil {
		return 0, 0, err
	}
	for _, p := range pending {
		if ctx.Err() != nil {
			return sent, failed, ctx.Err()
		}
		sendErr := s.channel.Send(ctx, notify.Message{ChatID: p.ChatID, Text: p.Payload})
		if sendErr != nil {
			failed++
			s.logger.Warn("send failed", "notification_id", p.ID, "err", sendErr)
			if err := s.repo.MarkFailed(ctx, p.ID); err != nil {
				s.logger.Error("mark failed", "notification_id", p.ID, "err", err)
			}
			continue
		}
		if err := s.repo.MarkSent(ctx, p.ID); err != nil {
			s.logger.Error("mark sent", "notification_id", p.ID, "err", err)
			continue
		}
		sent++
	}
	return sent, failed, nil
}

// Run sends on an interval until the context is cancelled.
func (s *NotifySender) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, _, err := s.RunOnce(ctx); err != nil && ctx.Err() == nil {
			s.logger.Error("notify send failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
