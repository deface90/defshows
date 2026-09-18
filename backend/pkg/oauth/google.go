package oauth

import (
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleOptions overrides endpoints for testing.
type GoogleOptions struct {
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	HTTPClient  *http.Client
}

// NewGoogle creates a Google OAuth provider.
func NewGoogle(clientID, clientSecret string, opts *GoogleOptions) Provider {
	endpoint := google.Endpoint
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo"
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
		name: "google",
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     endpoint,
			Scopes:       []string{"openid", "email", "profile"},
		},
		userInfoURL: userInfoURL,
		httpClient:  httpClient,
		parse: func(body []byte) (*UserInfo, error) {
			m, err := decodeMap(body)
			if err != nil {
				return nil, err
			}
			return &UserInfo{
				ProviderUserID: jsonString(m, "id"),
				Email:          jsonString(m, "email"),
				Name:           jsonString(m, "name"),
			}, nil
		},
	}
}
