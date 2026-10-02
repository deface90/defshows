package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	socialapi "github.com/deface90/defshows/backend/pkg/server/social"
)

// FeedCatalog resolves show and episode detail for feed-card enrichment.
type FeedCatalog interface {
	ShowByID(ctx context.Context, showID int64) (*entity.Show, error)
	Episodes(ctx context.Context, showID int64) ([]entity.Episode, error)
}

// FeedUserLookup resolves actor identity for home-feed cards.
type FeedUserLookup interface {
	FindUserByID(ctx context.Context, id int64) (*entity.User, error)
}

// SocialHandler implements the generated social ServerInterface: follows,
// approval of requests, follower/following listings, moderation reports, and the
// activity feeds.
type SocialHandler struct {
	social  *usecase.SocialUsecase
	feed    *usecase.FeedUsecase
	report  *usecase.ReportUsecase
	catalog FeedCatalog
	users   FeedUserLookup
}

// NewSocialHandler creates a SocialHandler.
func NewSocialHandler(socialUC *usecase.SocialUsecase, feedUC *usecase.FeedUsecase, reportUC *usecase.ReportUsecase, catalog FeedCatalog, users FeedUserLookup) *SocialHandler {
	return &SocialHandler{social: socialUC, feed: feedUC, report: reportUC, catalog: catalog, users: users}
}

var _ socialapi.ServerInterface = (*SocialHandler)(nil)

// FollowUser handles POST /me/follows/{userId}.
func (h *SocialHandler) FollowUser(c echo.Context, targetID socialapi.UserId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	f, err := h.social.Follow(c.Request().Context(), uid, targetID)
	switch {
	case errors.Is(err, usecase.ErrSelfFollow):
		return echo.NewHTTPError(http.StatusBadRequest, "cannot follow yourself")
	case errors.Is(err, usecase.ErrUserNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	case errors.Is(err, usecase.ErrBlocked):
		return echo.NewHTTPError(http.StatusForbidden, "blocked")
	case err != nil:
		return err
	}
	return c.JSON(http.StatusOK, socialapi.FollowResult{Status: socialapi.FollowResultStatus(f.Status)})
}

// BlockUser handles POST /me/blocks/{userId}.
func (h *SocialHandler) BlockUser(c echo.Context, targetID socialapi.UserId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	err := h.social.Block(c.Request().Context(), uid, targetID)
	switch {
	case errors.Is(err, usecase.ErrSelfBlock):
		return echo.NewHTTPError(http.StatusBadRequest, "cannot block yourself")
	case errors.Is(err, usecase.ErrUserNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	case err != nil:
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// UnblockUser handles DELETE /me/blocks/{userId}.
func (h *SocialHandler) UnblockUser(c echo.Context, targetID socialapi.UserId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.social.Unblock(c.Request().Context(), uid, targetID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// ListBlocks handles GET /me/blocks: the users the current user has blocked.
func (h *SocialHandler) ListBlocks(c echo.Context, params socialapi.ListBlocksParams) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	p := pageParams(params.Page, params.PageSize)
	users, total, err := h.social.Blocked(c.Request().Context(), uid, p.limit, p.offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toFollowUserList(users, total))
}

// UnfollowUser handles DELETE /me/follows/{userId}.
func (h *SocialHandler) UnfollowUser(c echo.Context, targetID socialapi.UserId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.social.Unfollow(c.Request().Context(), uid, targetID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// RemoveFollower handles DELETE /me/followers/{userId}.
func (h *SocialHandler) RemoveFollower(c echo.Context, targetID socialapi.UserId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.social.RemoveFollower(c.Request().Context(), uid, targetID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// CreateReport handles POST /me/reports: a user files a moderation report
// against another user.
func (h *SocialHandler) CreateReport(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	var req socialapi.ReportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	note := ""
	if req.Note != nil {
		note = *req.Note
	}
	rep, err := h.report.Report(c.Request().Context(), uid, req.TargetUserId, entity.ReportReason(req.Reason), note)
	switch {
	case errors.Is(err, usecase.ErrSelfReport):
		return echo.NewHTTPError(http.StatusBadRequest, "cannot report yourself")
	case errors.Is(err, usecase.ErrInvalidReason):
		return echo.NewHTTPError(http.StatusBadRequest, "invalid reason")
	case errors.Is(err, usecase.ErrInvalidNote):
		return echo.NewHTTPError(http.StatusBadRequest, "note too long")
	case errors.Is(err, usecase.ErrUserNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	case err != nil:
		return err
	}
	return c.JSON(http.StatusCreated, socialapi.Report{
		Id:           rep.ID,
		ReporterId:   rep.ReporterID,
		TargetUserId: rep.TargetUserID,
		Reason:       socialapi.ReportReason(rep.Reason),
		Note:         rep.Note,
		Status:       socialapi.ReportStatus(rep.Status),
		CreatedAt:    rep.CreatedAt,
	})
}

// ListIncomingRequests handles GET /me/follows/incoming.
func (h *SocialHandler) ListIncomingRequests(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	users, err := h.social.Incoming(c.Request().Context(), uid)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toFollowUserList(users, int64(len(users))))
}

// ApproveFollower handles POST /me/follows/incoming/{userId}/approve.
func (h *SocialHandler) ApproveFollower(c echo.Context, targetID socialapi.UserId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.social.Approve(c.Request().Context(), uid, targetID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "no pending request")
		}
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// RejectFollower handles POST /me/follows/incoming/{userId}/reject.
func (h *SocialHandler) RejectFollower(c echo.Context, targetID socialapi.UserId) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.social.Reject(c.Request().Context(), uid, targetID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "no pending request")
		}
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// ListFollowers handles GET /users/{userId}/followers.
func (h *SocialHandler) ListFollowers(c echo.Context, targetID socialapi.UserId, params socialapi.ListFollowersParams) error {
	return h.listEdge(c, targetID, pageParams(params.Page, params.PageSize), h.social.Followers)
}

// ListFollowing handles GET /users/{userId}/following.
func (h *SocialHandler) ListFollowing(c echo.Context, targetID socialapi.UserId, params socialapi.ListFollowingParams) error {
	return h.listEdge(c, targetID, pageParams(params.Page, params.PageSize), h.social.Following)
}

func (h *SocialHandler) listEdge(c echo.Context, targetID int64, p pageSpec, list func(ctx context.Context, userID int64, limit, offset int) ([]entity.User, int64, error)) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	ctx := c.Request().Context()
	canView, err := h.social.CanViewProfile(ctx, uid, targetID)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}
	if !canView {
		return echo.NewHTTPError(http.StatusForbidden, "profile is private")
	}
	users, total, err := list(ctx, targetID, p.limit, p.offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toFollowUserList(users, total))
}

func toFollowUserList(users []entity.User, total int64) socialapi.FollowUserList {
	out := make([]socialapi.FollowUser, 0, len(users))
	for i := range users {
		u := users[i]
		out = append(out, socialapi.FollowUser{
			Id:          u.ID,
			DisplayName: displayNameOf(&u),
			IsPublic:    u.IsPublic,
		})
	}
	return socialapi.FollowUserList{Users: out, Total: total}
}

type pageSpec struct{ limit, offset int }

// GetHomeFeed handles GET /me/feed: the aggregated activity of everyone the current
// user follows (accepted).
func (h *SocialHandler) GetHomeFeed(c echo.Context, params socialapi.GetHomeFeedParams) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	ctx := c.Request().Context()
	items, next, err := h.feed.HomeFeed(ctx, uid, strOr(params.Cursor), intOr(params.Limit))
	if err != nil {
		return err
	}
	page, err := h.toFeedPage(ctx, items, next, true)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, page)
}

// GetProfileFeed handles GET /users/{userId}/feed, gated by profile privacy.
func (h *SocialHandler) GetProfileFeed(c echo.Context, targetID socialapi.UserId, params socialapi.GetProfileFeedParams) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	ctx := c.Request().Context()
	canView, err := h.social.CanViewProfile(ctx, uid, targetID)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}
	if !canView {
		return echo.NewHTTPError(http.StatusForbidden, "profile is private")
	}
	items, next, err := h.feed.ProfileFeed(ctx, targetID, strOr(params.Cursor), intOr(params.Limit))
	if err != nil {
		return err
	}
	page, err := h.toFeedPage(ctx, items, next, false)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, page)
}

// toFeedPage enriches grouped feed items with show, episode-number and (for the home
// feed) actor detail, batching lookups by unique id.
func (h *SocialHandler) toFeedPage(ctx context.Context, items []usecase.FeedItem, next string, withActor bool) (socialapi.FeedPage, error) {
	showIDs := map[int64]struct{}{}
	actorIDs := map[int64]struct{}{}
	for _, it := range items {
		showIDs[it.ShowID] = struct{}{}
		if withActor {
			actorIDs[it.ActorID] = struct{}{}
		}
	}

	shows := make(map[int64]*entity.Show, len(showIDs))
	episodes := make(map[int64]socialapi.FeedEpisode)
	for id := range showIDs {
		s, err := h.catalog.ShowByID(ctx, id)
		if err != nil {
			return socialapi.FeedPage{}, err
		}
		shows[id] = s
		eps, err := h.catalog.Episodes(ctx, id)
		if err != nil {
			return socialapi.FeedPage{}, err
		}
		for i := range eps {
			e := eps[i]
			episodes[e.ID] = socialapi.FeedEpisode{SeasonNumber: e.SeasonNumber, EpisodeNumber: e.EpisodeNumber}
		}
	}

	actors := make(map[int64]*entity.User, len(actorIDs))
	for id := range actorIDs {
		u, err := h.users.FindUserByID(ctx, id)
		if err != nil {
			return socialapi.FeedPage{}, err
		}
		actors[id] = u
	}

	cards := make([]socialapi.FeedCard, 0, len(items))
	for i := range items {
		it := items[i]
		card := socialapi.FeedCard{
			Type:         socialapi.FeedCardType(it.Type),
			Count:        it.Count(),
			CreatedAt:    it.CreatedAt,
			SeasonNumber: it.Season,
			Rating:       it.Rating,
		}
		if s := shows[it.ShowID]; s != nil {
			card.Show = socialapi.FeedShow{Id: s.ID, TmdbId: s.TMDBID, Title: s.Title, PosterUrl: s.PosterKey}
		} else {
			card.Show = socialapi.FeedShow{Id: it.ShowID}
		}
		if it.Type == entity.EventWatchedEpisode {
			eps := make([]socialapi.FeedEpisode, 0, len(it.EpisodeIDs))
			for _, eid := range it.EpisodeIDs {
				if fe, ok := episodes[eid]; ok {
					eps = append(eps, fe)
				}
			}
			card.Episodes = &eps
		}
		if withActor {
			if u := actors[it.ActorID]; u != nil {
				card.Actor = &socialapi.FollowUser{Id: u.ID, DisplayName: displayNameOf(u), IsPublic: u.IsPublic}
			}
		}
		cards = append(cards, card)
	}

	page := socialapi.FeedPage{Cards: cards}
	if next != "" {
		page.NextCursor = &next
	}
	return page, nil
}

func strOr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func intOr(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func pageParams(page, pageSize *int) pageSpec {
	p, ps := 1, 20
	if page != nil {
		p = *page
	}
	if pageSize != nil {
		ps = *pageSize
	}
	if p < 1 {
		p = 1
	}
	if ps < 1 || ps > 100 {
		ps = 20
	}
	return pageSpec{limit: ps, offset: (p - 1) * ps}
}
