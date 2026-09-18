package oauth

import (
	"net/http"

	"golang.org/x/oauth2"
)

// YandexOptions overrides endpoints for testing.
type YandexOptions struct {
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	HTTPClient  *http.Client
}

// NewYandex creates a Yandex OAuth provider.
//
// NOTE: In production, login.yandex.ru/info also accepts "Authorization: OAuth
// <token>"; the standard Bearer header used here works for the common case.
func NewYandex(clientID, clientSecret string, opts *YandexOptions) Provider {
	endpoint := oauth2.Endpoint{
		AuthURL:  "https://oauth.yandex.ru/authorize",
		TokenURL: "https://oauth.yandex.ru/token",
	}
	userInfoURL := "https://login.yandex.ru/info?format=json"
	var httpClient *http.Client
	if opts != nil {
		if opts.AuthURL != "" {
			endpoint.AuthURL = opts.AuthURL
		}
		if opts.TokenURL != "" {
			endpoint.TokenURL = opts.TokenURL
		}
		if opts.UserInfoURL != "" {
			userInfoURL = opts.UserInfoURL
		}
		httpClient = opts.HTTPClient
	}
	return &oauth2Provider{
		name: "yandex",
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     endpoint,
			Scopes:       []string{"login:email", "login:info"},
		},
		userInfoURL: userInfoURL,
		httpClient:  httpClient,
		parse: func(body []byte) (*UserInfo, error) {
			m, err := decodeMap(body)
			if err != nil {
				return nil, err
			}
			name := jsonString(m, "real_name")
			if name == "" {
				name = jsonString(m, "display_name")
			}
			return &UserInfo{
				ProviderUserID: jsonString(m, "id"),
				Email:          jsonString(m, "default_email"),
				Name:           name,
			}, nil
		},
	}
}
