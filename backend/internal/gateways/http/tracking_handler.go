package httpapi

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/pkg/auth"
	trackingapi "github.com/deface90/defshows/backend/pkg/server/tracking"
)

// TrackingHandler implements the generated tracking ServerInterface.
type TrackingHandler struct {
	uc   *usecase.TrackingUsecase
	auth *usecase.AuthUsecase
}

// NewTrackingHandler creates a TrackingHandler.
func NewTrackingHandler(uc *usecase.TrackingUsecase, authUC *usecase.AuthUsecase) *TrackingHandler {
	return &TrackingHandler{uc: uc, auth: authUC}
}

var _ trackingapi.ServerInterface = (*TrackingHandler)(nil)

func userID(c echo.Context) (int64, bool) {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		return 0, false
	}
	return claims.UserID, true
}

// ListTracked handles GET /me/shows.
func (h *TrackingHandler) ListTracked(c echo.Context, params trackingapi.ListTrackedParams) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	status := ""
	if params.Status != nil {
		status = *params.Status
	}
	list, err := h.uc.ListShows(c.Request().Context(), uid, status)
	if err != nil {
		return err
	}
	out := make([]trackingapi.TrackedShow, 0, len(list))
	for i := range list {
		out = append(out, toAPITracked(list[i]))
	}
	return c.JSON(http.StatusOK, trackingapi.TrackedShowList{Tracked: out})
}

// AddShow handles POST /me/shows.
func (h *TrackingHandler) AddShow(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	var req trackingapi.AddShowRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	us, err := h.uc.AddShow(c.Request().Context(), uid, req.TmdbId)
	if err != nil {
		if errors.Is(err, usecase.ErrAlreadyTracked) {
			return echo.NewHTTPError(http.StatusConflict, "already tracked")
		}
		return echo.NewHTTPError(http.StatusBadGateway, "could not add show")
	}
	return c.JSON(http.StatusCreated, toAPIUserShow(us))
}

// GetTracked handles GET /me/shows/{showId}.
func (h *TrackingHandler) GetTracked(c echo.Context, showID trackingapi.ShowId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	ts, err := h.uc.GetTracked(c.Request().Context(), uid, showID)
	if err != nil {
		return trackingErr(err)
	}
	return c.JSON(http.StatusOK, toAPITracked(*ts))
}

// UpdateShow handles PATCH /me/shows/{showId}.
func (h *TrackingHandler) UpdateShow(c echo.Context, showID trackingapi.ShowId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	var req trackingapi.UpdateShowRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	var status *string
	if req.Status != nil {
		s := string(*req.Status)
		status = &s
	}
	us, err := h.uc.UpdateShow(c.Request().Context(), uid, showID, status, req.Favorite, req.PreferredDubbing)
	if err != nil {
		return trackingErr(err)
	}
	return c.JSON(http.StatusOK, toAPIUserShow(us))
}

// RemoveShow handles DELETE /me/shows/{showId}.
func (h *TrackingHandler) RemoveShow(c echo.Context, showID trackingapi.ShowId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.uc.RemoveShow(c.Request().Context(), uid, showID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// WatchEpisode handles POST /me/shows/{showId}/episodes/{episodeId}/watch.
func (h *TrackingHandler) WatchEpisode(c echo.Context, showID trackingapi.ShowId, episodeID trackingapi.EpisodeId) error {
	return h.setWatched(c, showID, episodeID, true)
}

// UnwatchEpisode handles DELETE /me/shows/{showId}/episodes/{episodeId}/watch.
func (h *TrackingHandler) UnwatchEpisode(c echo.Context, showID trackingapi.ShowId, episodeID trackingapi.EpisodeId) error {
	return h.setWatched(c, showID, episodeID, false)
}

func (h *TrackingHandler) setWatched(c echo.Context, showID, episodeID int64, watched bool) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.uc.SetEpisodeWatched(c.Request().Context(), uid, showID, episodeID, watched); err != nil {
		return trackingErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ListLinks handles GET /me/shows/{showId}/links.
func (h *TrackingHandler) ListLinks(c echo.Context, showID trackingapi.ShowId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	links, err := h.uc.ListLinks(c.Request().Context(), uid, showID)
	if err != nil {
		return trackingErr(err)
	}
	out := make([]trackingapi.Link, 0, len(links))
	for i := range links {
		out = append(out, toAPILink(&links[i]))
	}
	return c.JSON(http.StatusOK, trackingapi.LinkList{Links: out})
}

// AddLink handles POST /me/shows/{showId}/links.
func (h *TrackingHandler) AddLink(c echo.Context, showID trackingapi.ShowId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	var req trackingapi.AddLinkRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	label := ""
	if req.Label != nil {
		label = *req.Label
	}
	link, err := h.uc.AddLink(c.Request().Context(), uid, showID, string(req.Kind), label, req.Url)
	if err != nil {
		return trackingErr(err)
	}
	return c.JSON(http.StatusCreated, toAPILink(link))
}

// DeleteLink handles DELETE /me/shows/{showId}/links/{linkId}.
func (h *TrackingHandler) DeleteLink(c echo.Context, showID trackingapi.ShowId, linkID int64) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.uc.DeleteLink(c.Request().Context(), uid, showID, linkID); err != nil {
		return trackingErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// GetSettings handles GET /me/settings.
func (h *TrackingHandler) GetSettings(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	u, err := h.auth.Me(c.Request().Context(), uid)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	return c.JSON(http.StatusOK, trackingapi.Settings{Timezone: u.Timezone, IsPublic: u.IsPublic})
}

// UpdateSettings handles PATCH /me/settings.
func (h *TrackingHandler) UpdateSettings(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	var req trackingapi.UpdateSettingsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	ctx := c.Request().Context()
	u, err := h.auth.UpdateTimezone(ctx, uid, req.Timezone)
	if err != nil {
		return err
	}
	if req.IsPublic != nil {
		u, err = h.auth.SetProfileVisibility(ctx, uid, *req.IsPublic)
		if err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, trackingapi.Settings{Timezone: u.Timezone, IsPublic: u.IsPublic})
}

func trackingErr(err error) error {
	if errors.Is(err, usecase.ErrNotTracked) {
		return echo.NewHTTPError(http.StatusNotFound, "not tracked")
	}
	return err
}

func toAPIUserShow(us *entity.UserShow) trackingapi.UserShow {
	dub := us.PreferredDubbing
	return trackingapi.UserShow{
		Id:               us.ID,
		ShowId:           us.ShowID,
		Status:           trackingapi.UserShowStatus(us.Status),
		Favorite:         us.Favorite,
		PreferredDubbing: &dub,
	}
}

func toAPIShowRef(s *entity.Show) trackingapi.ShowRef {
	return trackingapi.ShowRef{
		Id:                 s.ID,
		TmdbId:             s.TMDBID,
		Title:              s.Title,
		OriginalTitle:      ptr(s.OriginalTitle),
		PosterUrl:          derefStr(s.PosterKey),
		AiringStatus:       trackingapi.ShowRefAiringStatus(s.AiringStatus),
		NextEpisodeAirDate: dateStr(s.NextEpisodeAirDate),
		VoteAverage:        f32(s.VoteAverage),
		VoteCount:          i64(s.VoteCount),
	}
}

func toAPIProgress(p usecase.Progress) trackingapi.Progress {
	prog := trackingapi.Progress{Watched: p.Watched, Total: p.Total, Unwatched: p.Unwatched, WatchedEpisodeIds: append([]int64{}, p.WatchedEpisodeIDs...)}
	if p.NextUnwatched != nil {
		id := p.NextUnwatched.ID
		prog.NextUnwatchedEpisodeId = &id
	}
	return prog
}

func toAPITracked(ts usecase.TrackedShow) trackingapi.TrackedShow {
	return trackingapi.TrackedShow{
		UserShow: toAPIUserShow(ts.UserShow),
		Show:     toAPIShowRef(ts.Show),
		Progress: toAPIProgress(ts.Progress),
	}
}

func toAPILink(l *entity.UserShowLink) trackingapi.Link {
	label := l.Label
	return trackingapi.Link{
		Id:    l.ID,
		Kind:  trackingapi.LinkKind(l.Kind),
		Label: &label,
		Url:   l.URL,
	}
}

// WatchShow handles POST /me/shows/{showId}/watch.
func (h *TrackingHandler) WatchShow(c echo.Context, showID trackingapi.ShowId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.uc.WatchShow(c.Request().Context(), uid, showID); err != nil {
		return trackingErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// UpdateLink handles PATCH /me/shows/{showId}/links/{linkId}.
func (h *TrackingHandler) UpdateLink(c echo.Context, showID trackingapi.ShowId, linkID int64) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	var req trackingapi.UpdateLinkRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := h.uc.UpdateLink(c.Request().Context(), uid, showID, linkID, req.Url); err != nil {
		return trackingErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}
