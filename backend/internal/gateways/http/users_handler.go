package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/pkg/auth"
	usersapi "github.com/deface90/defshows/backend/pkg/server/users"
)

// UsersHandler implements the generated users ServerInterface: a public
// directory plus read-only access to a user's tracked shows (respecting the
// profile's public/private flag).
type UsersHandler struct {
	auth     *usecase.AuthUsecase
	tracking *usecase.TrackingUsecase
}

// NewUsersHandler creates a UsersHandler.
func NewUsersHandler(authUC *usecase.AuthUsecase, trackingUC *usecase.TrackingUsecase) *UsersHandler {
	return &UsersHandler{auth: authUC, tracking: trackingUC}
}

var _ usersapi.ServerInterface = (*UsersHandler)(nil)

// ListUsers handles GET /users.
func (h *UsersHandler) ListUsers(c echo.Context) error {
	ctx := c.Request().Context()
	users, err := h.auth.ListUsers(ctx)
	if err != nil {
		return err
	}
	counts, err := h.tracking.ShowsCountByUser(ctx)
	if err != nil {
		return err
	}
	out := make([]usersapi.UserSummary, 0, len(users))
	for i := range users {
		u := users[i]
		out = append(out, usersapi.UserSummary{
			Id:          u.ID,
			DisplayName: displayNameOf(&u),
			Email:       u.Email,
			IsPublic:    u.IsPublic,
			ShowsCount:  counts[u.ID],
		})
	}
	return c.JSON(http.StatusOK, usersapi.UserList{Users: out})
}

// GetUserProfile handles GET /users/{userId}.
func (h *UsersHandler) GetUserProfile(c echo.Context, userID usersapi.UserId) error {
	ctx := c.Request().Context()
	u, err := h.auth.Me(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}
	counts, err := h.tracking.ShowsCountByUser(ctx)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, usersapi.UserProfile{
		Id:          u.ID,
		DisplayName: displayNameOf(u),
		IsPublic:    u.IsPublic,
		ShowsCount:  counts[u.ID],
	})
}

// ListUserShows handles GET /users/{userId}/shows. A private profile is only
// visible to its owner or an admin.
func (h *UsersHandler) ListUserShows(c echo.Context, userID usersapi.UserId) error {
	ctx := c.Request().Context()
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}

	target, err := h.auth.Me(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}

	isOwner := claims.UserID == target.ID
	isAdmin := claims.Role == string(entity.RoleAdmin)
	if !target.IsPublic && !isOwner && !isAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "profile is private")
	}

	list, err := h.tracking.ListShows(ctx, target.ID, "")
	if err != nil {
		return err
	}
	out := make([]usersapi.TrackedShow, 0, len(list))
	for i := range list {
		out = append(out, toUsersTracked(list[i]))
	}
	return c.JSON(http.StatusOK, usersapi.TrackedShowList{Tracked: out})
}

// displayNameOf returns a non-empty label for a user: their display name, else
// the local part of their email, else "Пользователь #<id>".
func displayNameOf(u *entity.User) string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	if u.Email != nil && *u.Email != "" {
		if at := strings.IndexByte(*u.Email, '@'); at > 0 {
			return (*u.Email)[:at]
		}
		return *u.Email
	}
	return fmt.Sprintf("Пользователь #%d", u.ID)
}

func toUsersUserShow(us *entity.UserShow) usersapi.UserShow {
	dub := us.PreferredDubbing
	return usersapi.UserShow{
		Id:               us.ID,
		ShowId:           us.ShowID,
		Status:           usersapi.UserShowStatus(us.Status),
		Favorite:         us.Favorite,
		PreferredDubbing: &dub,
	}
}

func toUsersShowRef(s *entity.Show) usersapi.ShowRef {
	return usersapi.ShowRef{
		Id:                 s.ID,
		TmdbId:             s.TMDBID,
		Title:              s.Title,
		OriginalTitle:      ptr(s.OriginalTitle),
		PosterUrl:          derefStr(s.PosterKey),
		AiringStatus:       usersapi.ShowRefAiringStatus(s.AiringStatus),
		NextEpisodeAirDate: dateStr(s.NextEpisodeAirDate),
		VoteAverage:        f32(s.VoteAverage),
		VoteCount:          i64(s.VoteCount),
	}
}

func toUsersProgress(p usecase.Progress) usersapi.Progress {
	prog := usersapi.Progress{Watched: p.Watched, Total: p.Total}
	if p.NextUnwatched != nil {
		id := p.NextUnwatched.ID
		prog.NextUnwatchedEpisodeId = &id
	}
	return prog
}

func toUsersTracked(ts usecase.TrackedShow) usersapi.TrackedShow {
	return usersapi.TrackedShow{
		UserShow: toUsersUserShow(ts.UserShow),
		Show:     toUsersShowRef(ts.Show),
		Progress: toUsersProgress(ts.Progress),
	}
}
