// Command notifier runs the defShows notification scanner, sender, and the
// Telegram bot update poller (for account linking).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/internal/service/workers"
	"github.com/deface90/defshows/backend/migrations"
	"github.com/deface90/defshows/backend/pkg/config"
	"github.com/deface90/defshows/backend/pkg/db"
	pkglog "github.com/deface90/defshows/backend/pkg/log"
	"github.com/deface90/defshows/backend/pkg/notify"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("notifier: config: %v", err)
	}
	logger := pkglog.New(cfg.Log)

	gdb, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("notifier: db: %v", err)
	}
	if err := db.RunMigrations(gdb, migrations.FS, migrations.Dir); err != nil {
		log.Fatalf("notifier: migrate: %v", err)
	}

	proxyTransport, err := telegramProxyTransport(cfg.Telegram.ProxyURL)
	if err != nil {
		log.Fatalf("notifier: telegram proxy: %v", err)
	}
	if cfg.Telegram.ProxyURL != "" {
		logger.Info("telegram traffic routed through proxy")
	}

	notificationRepo := repository.NewNotificationRepository(gdb)
	userRepo := repository.NewUserRepository(gdb)
	sendClient := &http.Client{Timeout: 10 * time.Second, Transport: proxyTransport}
	channel := notify.NewTelegramChannel(cfg.Telegram.BaseURL, cfg.Telegram.Token, sendClient)

	scanner := workers.NewNotifyScanner(notificationRepo, logger, cfg.Notifier.Lookback)
	sender := workers.NewNotifySender(notificationRepo, channel, logger, 100)
	linkUC := usecase.NewNotificationUsecase(notificationRepo, userRepo, cfg.Telegram.Username, cfg.Notifier.LinkTTL)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go scanner.Run(ctx, cfg.Notifier.ScanInterval)

	if cfg.Telegram.Token == "" {
		logger.Warn("TELEGRAM_BOT_TOKEN not set — sender and bot polling disabled")
	} else {
		go sender.Run(ctx, cfg.Notifier.SendInterval)
		pollClient := &http.Client{Timeout: 40 * time.Second, Transport: proxyTransport}
		go pollTelegram(ctx, logger, cfg.Telegram, pollClient, channel, linkUC)
	}

	logger.Info("notifier started")
	<-ctx.Done()
	logger.Info("notifier stopped")
}

// telegramProxyTransport returns an http RoundTripper routing through the given
// proxy URL (socks5:// or http(s)://), or the default transport when unset.
func telegramProxyTransport(proxyURL string) (http.RoundTripper, error) {
	if proxyURL == "" {
		return http.DefaultTransport, nil
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse TELEGRAM_PROXY_URL: %w", err)
	}
	return &http.Transport{Proxy: http.ProxyURL(u)}, nil
}

// pollTelegram long-polls getUpdates and links accounts on "/start <token>".
func pollTelegram(ctx context.Context, logger *slog.Logger, tg config.Telegram, client *http.Client, channel notify.Channel, linkUC *usecase.NotificationUsecase) {
	offset := 0
	for {
		if ctx.Err() != nil {
			return
		}
		updates, err := getUpdates(ctx, client, tg, offset)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Warn("getUpdates failed", "err", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			text := strings.TrimSpace(u.Message.Text)
			if !strings.HasPrefix(text, "/start ") {
				continue
			}
			token := strings.TrimSpace(strings.TrimPrefix(text, "/start "))
			chatID := u.Message.Chat.ID
			reply := "Аккаунт привязан! Будут приходить уведомления о новых сериях."
			if err := linkUC.LinkTelegram(ctx, token, chatID); err != nil {
				logger.Warn("telegram link failed", "err", err)
				reply = "Ссылка недействительна или истекла. Запросите новую в defShows."
			}
			_ = channel.Send(ctx, notify.Message{ChatID: chatID, Text: reply})
		}
	}
}

type tgUpdate struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

func getUpdates(ctx context.Context, client *http.Client, tg config.Telegram, offset int) ([]tgUpdate, error) {
	endpoint := fmt.Sprintf("%s/bot%s/getUpdates?timeout=30&offset=%d", tg.BaseURL, tg.Token, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getUpdates status %d", resp.StatusCode)
	}
	var body struct {
		OK     bool       `json:"ok"`
		Result []tgUpdate `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Result, nil
}
