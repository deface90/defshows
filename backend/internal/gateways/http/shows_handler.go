package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	showsapi "github.com/deface90/defshows/backend/pkg/server/shows"
)

// ShowsHandler implements the generated shows ServerInterface.
type ShowsHandler struct {
	uc *usecase.CatalogUsecase
}

// NewShowsHandler creates a ShowsHandler.
func NewShowsHandler(uc *usecase.CatalogUsecase) *ShowsHandler {
	return &ShowsHandler{uc: uc}
}

var _ showsapi.ServerInterface = (*ShowsHandler)(nil)

// SearchShows handles GET /shows/search.
func (h *ShowsHandler) SearchShows(c echo.Context, params showsapi.SearchShowsParams) error {
	res, err := h.uc.SearchExternal(c.Request().Context(), params.Q)
	if err != nil {
		slog.ErrorContext(c.Request().Context(), "provider search failed", "provider", "tmdb", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "provider search failed")
	}
	return c.JSON(http.StatusOK, showsapi.SearchResults{Results: toAPISummaries(res)})
}

// ImportShow handles POST /shows/import.
func (h *ShowsHandler) ImportShow(c echo.Context) error {
	var req showsapi.ImportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	show, err := h.uc.ImportShow(c.Request().Context(), req.TmdbId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "import failed")
	}
	detail, err := h.uc.GetShow(c.Request().Context(), show.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toAPIShowDetail(detail))
}

// ListShows handles GET /shows.
func (h *ShowsHandler) ListShows(c echo.Context, params showsapi.ListShowsParams) error {
	q := ""
	if params.Q != nil {
		q = *params.Q
	}
	shows, err := h.uc.SearchLocal(c.Request().Context(), q, 50, 0)
	if err != nil {
		return err
	}
	out := make([]showsapi.Show, 0, len(shows))
	for i := range shows {
		out = append(out, toAPIShowBasic(&shows[i]))
	}
	return c.JSON(http.StatusOK, showsapi.ShowList{Shows: out})
}

// GetShow handles GET /shows/{id}.
func (h *ShowsHandler) GetShow(c echo.Context, id int64) error {
	detail, err := h.uc.GetShow(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrShowNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "show not found")
		}
		return err
	}
	return c.JSON(http.StatusOK, toAPIShowDetail(detail))
}

func toAPIShowBasic(s *entity.Show) showsapi.Show {
	return showsapi.Show{
		Id:                 s.ID,
		TmdbId:             s.TMDBID,
		Title:              s.Title,
		OriginalTitle:      ptr(s.OriginalTitle),
		Overview:           ptr(s.Overview),
		PosterUrl:          derefStr(s.PosterKey),
		BackdropUrl:        derefStr(s.BackdropKey),
		Status:             ptr(s.Status),
		InProduction:       ptrBool(s.InProduction),
		FirstAirDate:       dateStr(s.FirstAirDate),
		LastAirDate:        dateStr(s.LastAirDate),
		NextEpisodeAirDate: dateStr(s.NextEpisodeAirDate),
		AiringStatus:       showsapi.ShowAiringStatus(s.AiringStatus),
		OriginalLanguage:   ptr(s.OriginalLanguage),
		Popularity:         f32(s.Popularity),
		VoteAverage:        f32(s.VoteAverage),
		VoteCount:          i64(s.VoteCount),
	}
}

func toAPIShowDetail(d *usecase.ShowDetail) showsapi.Show {
	show := toAPIShowBasic(d.Show)

	genres := make([]showsapi.Genre, 0, len(d.Show.Genres))
	for _, g := range d.Show.Genres {
		g := g
		genres = append(genres, showsapi.Genre{Id: g.ID, TmdbId: &g.TMDBID, Name: g.Name})
	}
	show.Genres = &genres

	ratings := make([]showsapi.Rating, 0, len(d.Ratings))
	for _, r := range d.Ratings {
		r := r
		ratings = append(ratings, showsapi.Rating{Source: r.Source, Value: r.Value, Votes: r.Votes})
	}
	show.Ratings = &ratings

	episodesBySeason := map[int][]showsapi.Episode{}
	for _, e := range d.Episodes {
		e := e
		episodesBySeason[e.SeasonNumber] = append(episodesBySeason[e.SeasonNumber], showsapi.Episode{
			Id:            e.ID,
			SeasonNumber:  e.SeasonNumber,
			EpisodeNumber: e.EpisodeNumber,
			Name:          e.Name,
			Overview:      ptr(e.Overview),
			AirDate:       dateStr(e.AirDate),
			Runtime:       e.Runtime,
			StillUrl:      derefStr(e.StillKey),
		})
	}

	seasons := make([]showsapi.Season, 0, len(d.Show.Seasons))
	for _, s := range d.Show.Seasons {
		s := s
		eps := episodesBySeason[s.SeasonNumber]
		if eps == nil {
			eps = []showsapi.Episode{}
		}
		count := s.EpisodeCount
		seasons = append(seasons, showsapi.Season{
			Id:           s.ID,
			SeasonNumber: s.SeasonNumber,
			Name:         s.Name,
			Overview:     ptr(s.Overview),
			AirDate:      dateStr(s.AirDate),
			EpisodeCount: &count,
			PosterUrl:    derefStr(s.PosterKey),
			Episodes:     eps,
		})
	}
	show.Seasons = &seasons
	return show
}

// --- small mapping helpers ---

func ptr(s string) *string { return &s }

func ptrBool(b bool) *bool { return &b }

func derefStr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func f32(v float64) *float32 {
	f := float32(v)
	return &f
}

func i64(v int64) *int64 { return &v }

func dateStr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

// GetShowByTMDB loads catalog details without adding the show to a user's list.
func (h *ShowsHandler) GetShowByTMDB(c echo.Context, tmdbID int64) error {
	if tmdbID <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid TMDB ID")
	}
	show, err := h.uc.EnsureShow(c.Request().Context(), tmdbID)
	if err != nil {
		slog.ErrorContext(c.Request().Context(), "provider show detail failed", "tmdb_id", tmdbID, "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "provider show detail failed")
	}
	return h.GetShow(c, show.ID)
}
