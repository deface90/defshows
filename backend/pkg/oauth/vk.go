package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// VKOptions overrides endpoints for testing.
type VKOptions struct {
	AuthURL    string
	TokenURL   string
	HTTPClient *http.Client
}

// vkProvider implements VK OAuth. VK returns the user id and (with the email
// scope) the email directly in the token response, so no separate userinfo call
// is needed. Display name is left empty (fetching it requires an extra
// api.vk.com call and is optional for defShows).
type vkProvider struct {
	clientID     string
	clientSecret string
	authURL      string
	tokenURL     string
	httpClient   *http.Client
}

// NewVK creates a VK OAuth provider.
func NewVK(clientID, clientSecret string, opts *VKOptions) Provider {
	p := &vkProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		authURL:      "https://oauth.vk.com/authorize",
		tokenURL:     "https://oauth.vk.com/access_token",
		httpClient:   http.DefaultClient,
	}
	if opts != nil {
		if opts.AuthURL != "" {
			p.authURL = opts.AuthURL
		}
		if opts.TokenURL != "" {
			p.tokenURL = opts.TokenURL
		}
		if opts.HTTPClient != nil {
			p.httpClient = opts.HTTPClient
		}
	}
	return p
}

func (p *vkProvider) Name() string { return "vk" }

func (p *vkProvider) AuthURL(state, redirectURI string) string {
	q := url.Values{}
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "email")
	q.Set("state", state)
	q.Set("v", "5.131")
	return p.authURL + "?" + q.Encode()
}

func (p *vkProvider) Exchange(ctx context.Context, code, redirectURI string) (*UserInfo, error) {
	q := url.Values{}
	q.Set("client_id", p.clientID)
	q.Set("client_secret", p.clientSecret)
	q.Set("redirect_uri", redirectURI)
	q.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.tokenURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: vk exchange: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth: vk exchange: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		UserID      int64  `json:"user_id"`
		Email       string `json:"email"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, err
	}
	if tr.Error != "" || tr.UserID == 0 {
		return nil, fmt.Errorf("oauth: vk exchange: %s", tr.Error)
	}
	return &UserInfo{
		ProviderUserID: strconv.FormatInt(tr.UserID, 10),
		Email:          tr.Email, // may be empty if the user denied the email scope
	}, nil
}
