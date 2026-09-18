// Package oauth defines the social login provider interface and its
// implementations (Google, Yandex, VK).
package oauth

import "context"

// UserInfo is the normalized identity returned by a provider.
type UserInfo struct {
	ProviderUserID string
	Email          string
	Name           string
}

// Provider is a social login provider.
type Provider interface {
	// Name returns the provider key (e.g. "google").
	Name() string
	// AuthURL builds the authorization redirect URL.
	AuthURL(state, redirectURI string) string
	// Exchange swaps an authorization code for the user's identity.
	Exchange(ctx context.Context, code, redirectURI string) (*UserInfo, error)
}

// Registry maps provider names to implementations.
type Registry map[string]Provider

// NewRegistry builds a Registry from providers, skipping nil entries.
func NewRegistry(providers ...Provider) Registry {
	r := Registry{}
	for _, p := range providers {
		if p != nil {
			r[p.Name()] = p
		}
	}
	return r
}

// Get returns a provider by name.
func (r Registry) Get(name string) (Provider, bool) {
	p, ok := r[name]
	return p, ok
}
