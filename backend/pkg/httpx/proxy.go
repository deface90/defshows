// Package httpx holds small HTTP helpers shared across services.
package httpx

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Transport returns an http RoundTripper that routes requests through the given
// proxy URL (socks5:// or http(s)://), or http.DefaultTransport when the URL is
// empty.
func Transport(proxyURL string) (http.RoundTripper, error) {
	if proxyURL == "" {
		return http.DefaultTransport, nil
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse proxy url: %w", err)
	}
	return &http.Transport{Proxy: http.ProxyURL(u)}, nil
}

// Client returns an *http.Client with the given timeout whose transport routes
// through the proxy URL when set (otherwise the default transport).
func Client(proxyURL string, timeout time.Duration) (*http.Client, error) {
	t, err := Transport(proxyURL)
	if err != nil {
		return nil, err
	}
	return &http.Client{Timeout: timeout, Transport: t}, nil
}
