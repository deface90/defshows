package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
)

// fcmScope is the OAuth2 scope required to call the FCM HTTP v1 API.
const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

// FCMChannel sends messages via Firebase Cloud Messaging's HTTP v1 API,
// authenticating as a service account. The oauth2 client returned by
// JWTConfigFromJSON caches and refreshes its access token automatically, so
// no manual token bookkeeping is needed (unlike APNsChannel's hand-rolled JWT).
type FCMChannel struct {
	projectID string
	client    *http.Client
}

var _ Channel = (*FCMChannel)(nil)

// NewFCMChannel creates an FCMChannel. serviceAccountJSON is the full
// contents of a Firebase service account key (Project Settings -> Service
// Accounts -> Generate new private key).
func NewFCMChannel(serviceAccountJSON, projectID string) (*FCMChannel, error) {
	cfg, err := google.JWTConfigFromJSON([]byte(serviceAccountJSON), fcmScope)
	if err != nil {
		return nil, fmt.Errorf("fcm: parse service account: %w", err)
	}
	client := cfg.Client(context.Background())
	client.Timeout = 10 * time.Second
	return &FCMChannel{projectID: projectID, client: client}, nil
}

// Send delivers a message to a single device. msg.Target is the FCM
// registration token.
func (f *FCMChannel) Send(ctx context.Context, msg Message) error {
	payload := map[string]any{
		"message": map[string]any{
			"token":        msg.Target,
			"notification": map[string]string{"title": msg.Title, "body": msg.Body},
			"data":         msg.Data,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", f.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("fcm: send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var errBody struct {
		Error struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if resp.StatusCode == http.StatusNotFound || strings.Contains(errBody.Error.Message, "UNREGISTERED") {
		return &UnregisteredTargetError{Target: msg.Target}
	}
	return fmt.Errorf("fcm: send: status %d status %s message %s", resp.StatusCode, errBody.Error.Status, errBody.Error.Message)
}
