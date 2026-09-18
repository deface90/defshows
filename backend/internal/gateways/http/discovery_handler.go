package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/pkg/provider"
	showsapi "github.com/deface90/defshows/backend/pkg/server/shows"
	"github.com/labstack/echo/v4"
)

func (h *ShowsHandler) DiscoverShows(c echo.Context, params showsapi.DiscoverShowsParams) error {
	opts := provider.DiscoverOptions{Page: 1, RatingMax: 10, Sort: "popularity.desc"}
	if params.Country != nil {
		opts.Country = *params.Country
	}
	if params.Genres != nil {
		opts.Genres = *params.Genres
	}
	if params.DateFrom != nil {
		opts.DateFrom = *params.DateFrom
	}
	if params.DateTo != nil {
		opts.DateTo = *params.DateTo
	}
	if params.RatingMin != nil {
		opts.RatingMin = float64(*params.RatingMin)
	}
	if params.RatingMax != nil {
		opts.RatingMax = float64(*params.RatingMax)
	}
	if params.VotesMin != nil {
		opts.VotesMin = *params.VotesMin
	}
	if params.Page != nil {
		opts.Page = *params.Page
	}
	if params.Sort != nil {
		opts.Sort = string(*params.Sort)
	}
	page, err := h.uc.Discover(c.Request().Context(), opts)
	if errors.Is(err, usecase.ErrInvalidDiscoveryFilters) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid discovery filters")
	}
	if err != nil {
		slog.ErrorContext(c.Request().Context(), "provider discovery failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "provider discovery failed")
	}
	return c.JSON(http.StatusOK, showsapi.DiscoveryResults{Results: toAPISummaries(page.Results), Page: page.Page, TotalPages: page.TotalPages, TotalResults: page.TotalResults})
}

func (h *ShowsHandler) GetDiscoveryFilters(c echo.Context) error {
	filters, err := h.uc.DiscoveryFilters(c.Request().Context())
	if err != nil {
		slog.ErrorContext(c.Request().Context(), "provider discovery filters failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "provider discovery filters failed")
	}
	out := showsapi.DiscoveryFilters{}
	out.Genres = make([]struct {
		Id   int64  `json:"id"`
		Name string `json:"name"`
	}, len(filters.Genres))
	out.Countries = make([]struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}, len(filters.Countries))
	for i, g := range filters.Genres {
		out.Genres[i].Id, out.Genres[i].Name = g.TMDBID, g.Name
	}
	for i, country := range filters.Countries {
		out.Countries[i].Code, out.Countries[i].Name = country.Code, country.Name
	}
	return c.JSON(http.StatusOK, out)
}

func toAPISummaries(results []provider.ShowSummary) []showsapi.ShowSummary {
	out := make([]showsapi.ShowSummary, 0, len(results))
	for _, s := range results {
		out = append(out, showsapi.ShowSummary{TmdbId: s.TMDBID, Title: s.Title, OriginalTitle: ptr(s.OriginalTitle), Overview: ptr(s.Overview), PosterUrl: ptr(s.PosterURL), FirstAirDate: dateStr(s.FirstAirDate), Popularity: f32(s.Popularity), VoteAverage: f32(s.VoteAverage), VoteCount: i64(s.VoteCount)})
	}
	return out
}
