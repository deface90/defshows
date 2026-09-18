package httpapi

import (
	"github.com/deface90/defshows/backend/pkg/config"
	"github.com/deface90/defshows/backend/pkg/oauth"
)

// BuildOAuthRegistry constructs the OAuth provider registry from config,
// including only providers that are configured.
func BuildOAuthRegistry(cfg config.OAuth) oauth.Registry {
	var providers []oauth.Provider
	if cfg.Google.Enabled() {
		providers = append(providers, oauth.NewGoogle(cfg.Google.ClientID, cfg.Google.ClientSecret, nil))
	}
	if cfg.Yandex.Enabled() {
		providers = append(providers, oauth.NewYandex(cfg.Yandex.ClientID, cfg.Yandex.ClientSecret, nil))
	}
	if cfg.VK.Enabled() {
		providers = append(providers, oauth.NewVK(cfg.VK.ClientID, cfg.VK.ClientSecret, nil))
	}
	return oauth.NewRegistry(providers...)
}
