package usecase_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/pkg/provider"
)

type discoveryProvider struct {
	fakeProvider
	calls int
}

func (p *discoveryProvider) Discover(context.Context, provider.DiscoverOptions) (provider.DiscoveryPage, error) {
	p.calls++
	return provider.DiscoveryPage{}, nil
}
func (p *discoveryProvider) DiscoveryFilters(context.Context) (provider.DiscoveryFilters, error) {
	return provider.DiscoveryFilters{}, nil
}

func TestDiscoveryValidation(t *testing.T) {
	tests := map[string]func(*provider.DiscoverOptions){
		"page":       func(o *provider.DiscoverOptions) { o.Page = 501 },
		"votes":      func(o *provider.DiscoverOptions) { o.VotesMin = -1 },
		"rating":     func(o *provider.DiscoverOptions) { o.RatingMin = 11 },
		"nan":        func(o *provider.DiscoverOptions) { o.RatingMin = math.NaN() },
		"country":    func(o *provider.DiscoverOptions) { o.Country = "US|GB" },
		"genres":     func(o *provider.DiscoverOptions) { o.Genres = "18|abc" },
		"date":       func(o *provider.DiscoverOptions) { o.DateFrom = "2020-02-31" },
		"date order": func(o *provider.DiscoverOptions) { o.DateFrom = "2021-01-01"; o.DateTo = "2020-01-01" },
		"sort":       func(o *provider.DiscoverOptions) { o.Sort = "other" },
	}
	for name, modify := range tests {
		t.Run(name, func(t *testing.T) {
			p := &discoveryProvider{}
			uc := usecase.NewCatalogUsecase(nil, p)
			opts := provider.DiscoverOptions{Page: 1, RatingMax: 10, Sort: "popularity.desc"}
			modify(&opts)
			if _, err := uc.Discover(t.Context(), opts); !errors.Is(err, usecase.ErrInvalidDiscoveryFilters) {
				t.Fatalf("error: %v", err)
			}
			if p.calls != 0 {
				t.Fatal("invalid request reached upstream")
			}
		})
	}
}
