package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// oauth2Provider is a generic OAuth2 authorization-code provider parameterized
// by endpoints and a userinfo parser. Google/Yandex/VK are built on top of it.
type oauth2Provider struct {
	name        string
	cfg         *oauth2.Config
	userInfoURL string
	parse       func([]byte) (*UserInfo, error)
	httpClient  *http.Client
}

func (p *oauth2Provider) Name() string { return p.name }

func (p *oauth2Provider) AuthURL(state, redirectURI string) string {
	c := *p.cfg
	c.RedirectURL = redirectURI
	return c.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *oauth2Provider) Exchange(ctx context.Context, code, redirectURI string) (*UserInfo, error) {
	if p.httpClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, p.httpClient)
	}
	c := *p.cfg
	c.RedirectURL = redirectURI

	tok, err := c.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oauth: %s exchange: %w", p.name, err)
	}

	resp, err := c.Client(ctx, tok).Get(p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("oauth: %s userinfo: %w", p.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth: %s userinfo: status %d", p.name, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return p.parse(body)
}

// jsonString safely extracts a string field.
func jsonString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func decodeMap(body []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}
