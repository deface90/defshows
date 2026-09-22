package provider

import "context"

// DiscoveryProvider is the optional catalog browsing capability.
type DiscoveryProvider interface {
	Discover(context.Context, DiscoverOptions) (DiscoveryPage, error)
	DiscoveryFilters(context.Context) (DiscoveryFilters, error)
	Trending(context.Context) (DiscoveryPage, error)
}

type DiscoverOptions struct {
	Country, Genres, DateFrom, DateTo, Sort string
	RatingMin, RatingMax                    float64
	VotesMin, Page                          int
}

type DiscoveryPage struct {
	Results                        []ShowSummary
	Page, TotalPages, TotalResults int
}

type Country struct{ Code, Name string }
type DiscoveryFilters struct {
	Genres    []Genre
	Countries []Country
}
