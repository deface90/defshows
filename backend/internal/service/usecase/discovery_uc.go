package usecase

import (
	"context"
	"errors"
	"math"
	"regexp"
	"time"

	"github.com/deface90/defshows/backend/pkg/provider"
)

var ErrInvalidDiscoveryFilters = errors.New("invalid discovery filters")
var genreFilterPattern = regexp.MustCompile(`^[1-9][0-9]*(?:,[1-9][0-9]*)*$|^[1-9][0-9]*(?:\|[1-9][0-9]*)*$`)
var countryFilterPattern = regexp.MustCompile(`^[A-Z]{2}$`)

func (uc *CatalogUsecase) Discover(ctx context.Context, opts provider.DiscoverOptions) (provider.DiscoveryPage, error) {
	if opts.Page < 1 || opts.Page > 500 || opts.VotesMin < 0 || opts.RatingMin < 0 || opts.RatingMax > 10 || opts.RatingMin > opts.RatingMax || math.IsNaN(opts.RatingMin) || math.IsNaN(opts.RatingMax) || math.IsInf(opts.RatingMin, 0) || math.IsInf(opts.RatingMax, 0) {
		return provider.DiscoveryPage{}, ErrInvalidDiscoveryFilters
	}
	if opts.Country != "" && !countryFilterPattern.MatchString(opts.Country) || len(opts.Genres) > 500 || opts.Genres != "" && !genreFilterPattern.MatchString(opts.Genres) {
		return provider.DiscoveryPage{}, ErrInvalidDiscoveryFilters
	}
	for _, date := range []string{opts.DateFrom, opts.DateTo} {
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return provider.DiscoveryPage{}, ErrInvalidDiscoveryFilters
			}
		}
	}
	if opts.DateFrom != "" && opts.DateTo != "" && opts.DateFrom > opts.DateTo {
		return provider.DiscoveryPage{}, ErrInvalidDiscoveryFilters
	}
	switch opts.Sort {
	case "popularity.desc", "vote_average.desc", "first_air_date.desc":
	default:
		return provider.DiscoveryPage{}, ErrInvalidDiscoveryFilters
	}
	p, ok := uc.provider.(provider.DiscoveryProvider)
	if !ok {
		return provider.DiscoveryPage{}, errors.New("provider does not support discovery")
	}
	return p.Discover(ctx, opts)
}

func (uc *CatalogUsecase) DiscoveryFilters(ctx context.Context) (provider.DiscoveryFilters, error) {
	p, ok := uc.provider.(provider.DiscoveryProvider)
	if !ok {
		return provider.DiscoveryFilters{}, errors.New("provider does not support discovery")
	}
	return p.DiscoveryFilters(ctx)
}
