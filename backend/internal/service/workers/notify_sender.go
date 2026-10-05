package workers

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/pkg/notify"
)

// SenderRepo is the storage dependency of NotifySender.
type SenderRepo interface {
	Pending(ctx context.Context, channel string, limit int) ([]repository.PendingNotification, error)
	MarkSent(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64) error
	// ClearTarget drops a now-invalid delivery target (device token / chat id) for the
	// channel so it stops being used; the device re-registers a fresh one on next launch.
	ClearTarget(ctx context.Context, channel, target string) error
}

// NotifySender delivers a channel's pending notifications.
type NotifySender struct {
	repo        SenderRepo
	channelName string
	channel     notify.Channel
	logger      *slog.Logger
	batch       int
}

// NewNotifySender creates a NotifySender for one channel ("telegram" or
// "apns").
func NewNotifySender(repo SenderRepo, channelName string, channel notify.Channel, logger *slog.Logger, batch int) *NotifySender {
	if batch <= 0 {
		batch = 100
	}
	return &NotifySender{repo: repo, channelName: channelName, channel: channel, logger: logger, batch: batch}
}

// RunOnce sends one batch of pending notifications.
func (s *NotifySender) RunOnce(ctx context.Context) (sent, failed int, err error) {
	pending, err := s.repo.Pending(ctx, s.channelName, s.batch)
	if err != nil {
		return 0, 0, err
	}
	for _, p := range pending {
		if ctx.Err() != nil {
			return sent, failed, ctx.Err()
		}
		var data map[string]string
		if p.ShowID != nil {
			data = map[string]string{"show_id": strconv.FormatInt(*p.ShowID, 10)}
		}
		sendErr := s.channel.Send(ctx, notify.Message{Target: p.Target, Title: p.Title, Body: p.Body, Data: data})
		if sendErr != nil {
			failed++
			s.logger.Warn("send failed", "delivery_id", p.ID, "err", sendErr)
			var unreg *notify.UnregisteredTargetError
			if errors.As(sendErr, &unreg) {
				// Target is permanently invalid (app removed / token invalidated): drop it so
				// we stop retrying. The device re-registers a fresh token on next launch.
				if err := s.repo.ClearTarget(ctx, s.channelName, p.Target); err != nil {
					s.logger.Error("clear dead target", "delivery_id", p.ID, "err", err)
				}
			}
			if err := s.repo.MarkFailed(ctx, p.ID); err != nil {
				s.logger.Error("mark failed", "delivery_id", p.ID, "err", err)
			}
			continue
		}
		if err := s.repo.MarkSent(ctx, p.ID); err != nil {
			s.logger.Error("mark sent", "delivery_id", p.ID, "err", err)
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
