package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TelegramChannel sends messages via the Telegram Bot API.
type TelegramChannel struct {
	baseURL string
	token   string
	client  *http.Client
}

var _ Channel = (*TelegramChannel)(nil)

// NewTelegramChannel creates a Telegram channel. baseURL defaults to the public
// Bot API; override for testing.
func NewTelegramChannel(baseURL, token string, client *http.Client) *TelegramChannel {
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &TelegramChannel{baseURL: baseURL, token: token, client: client}
}

// Send delivers a message via sendMessage.
func (t *TelegramChannel) Send(ctx context.Context, msg Message) error {
	body, err := json.Marshal(map[string]any{
		"chat_id": msg.ChatID,
		"text":    msg.Text,
	})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/bot%s/sendMessage", t.baseURL, t.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: send: status %d", resp.StatusCode)
	}
	return nil
}
