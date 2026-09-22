package notify

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// APNsChannel sends messages via Apple's HTTP/2 APNs API, authenticating with
// a token-based (.p8) provider key. Go's net/http negotiates HTTP/2
// automatically for https requests, so no extra transport setup is needed.
type APNsChannel struct {
	host   string // api.push.apple.com or api.sandbox.push.apple.com
	topic  string // bundle id
	teamID string
	keyID  string
	key    *ecdsa.PrivateKey
	client *http.Client

	mu       sync.Mutex
	token    string
	tokenIat time.Time
}

var _ Channel = (*APNsChannel)(nil)

// NewAPNsChannel creates an APNsChannel. keyPEM is the contents of the .p8
// provider auth key. production selects api.push.apple.com over the sandbox
// gateway.
func NewAPNsChannel(teamID, keyID, keyPEM, topic string, production bool, client *http.Client) (*APNsChannel, error) {
	key, err := jwt.ParseECPrivateKeyFromPEM([]byte(keyPEM))
	if err != nil {
		return nil, fmt.Errorf("apns: parse key: %w", err)
	}
	host := "api.sandbox.push.apple.com"
	if production {
		host = "api.push.apple.com"
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &APNsChannel{host: host, topic: topic, teamID: teamID, keyID: keyID, key: key, client: client}, nil
}

// Send delivers a message via a single-device APNs request. msg.Target is the
// device token (hex).
func (a *APNsChannel) Send(ctx context.Context, msg Message) error {
	token, err := a.authToken()
	if err != nil {
		return fmt.Errorf("apns: auth token: %w", err)
	}

	payload := map[string]any{
		"aps": map[string]any{
			"alert": map[string]string{"title": msg.Title, "body": msg.Body},
			"sound": "default",
		},
	}
	for k, v := range msg.Data {
		payload[k] = v
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s/3/device/%s", a.host, msg.Target)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "bearer "+token)
	req.Header.Set("apns-topic", a.topic)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("content-type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("apns: send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var reason struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&reason)
	if resp.StatusCode == http.StatusGone || reason.Reason == "Unregistered" || reason.Reason == "BadDeviceToken" {
		return &UnregisteredTargetError{Target: msg.Target}
	}
	return fmt.Errorf("apns: send: status %d reason %s", resp.StatusCode, reason.Reason)
}

// UnregisteredTargetError signals that a target (e.g. device token) is no
// longer valid and should stop being used.
type UnregisteredTargetError struct {
	Target string
}

func (e *UnregisteredTargetError) Error() string {
	return fmt.Sprintf("apns: target %q unregistered", e.Target)
}

// authToken returns a cached provider auth JWT, regenerating it if it's
// older than 50 minutes (APNs tokens are valid up to an hour; Apple asks
// clients not to regenerate on every request).
func (a *APNsChannel) authToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.token != "" && time.Since(a.tokenIat) < 50*time.Minute {
		return a.token, nil
	}
	claims := jwt.MapClaims{"iss": a.teamID, "iat": time.Now().Unix()}
	t := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	t.Header["kid"] = a.keyID
	signed, err := t.SignedString(a.key)
	if err != nil {
		return "", err
	}
	a.token = signed
	a.tokenIat = time.Now()
	return a.token, nil
}
