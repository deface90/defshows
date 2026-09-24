package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	notificationsapi "github.com/deface90/defshows/backend/pkg/server/notifications"
)

// NotificationsHandler implements the generated notifications ServerInterface.
type NotificationsHandler struct {
	uc *usecase.NotificationUsecase
}

// NewNotificationsHandler creates a NotificationsHandler.
func NewNotificationsHandler(uc *usecase.NotificationUsecase) *NotificationsHandler {
	return &NotificationsHandler{uc: uc}
}

var _ notificationsapi.ServerInterface = (*NotificationsHandler)(nil)

// ListNotifications handles GET /me/notifications.
func (h *NotificationsHandler) ListNotifications(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	feed, err := h.uc.Feed(c.Request().Context(), uid, 50)
	if err != nil {
		return err
	}
	out := make([]notificationsapi.NotificationItem, 0, len(feed))
	for i := range feed {
		out = append(out, toAPINotification(&feed[i]))
	}
	return c.JSON(http.StatusOK, notificationsapi.NotificationFeed{Notifications: out})
}

// MarkNotificationRead handles POST /me/notifications/{id}/read.
func (h *NotificationsHandler) MarkNotificationRead(c echo.Context, id int64) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.uc.MarkRead(c.Request().Context(), uid, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "notification not found")
		}
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// MarkAllNotificationsRead handles POST /me/notifications/read-all.
func (h *NotificationsHandler) MarkAllNotificationsRead(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.uc.MarkAllRead(c.Request().Context(), uid); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// GetTelegramStatus handles GET /me/telegram.
func (h *NotificationsHandler) GetTelegramStatus(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	linked, err := h.uc.IsTelegramLinked(c.Request().Context(), uid)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, notificationsapi.TelegramStatus{Linked: linked})
}

// GetPrefs handles GET /me/notifications/prefs.
func (h *NotificationsHandler) GetPrefs(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	p, err := h.uc.GetPrefs(c.Request().Context(), uid)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toAPIPrefs(p))
}

// UpdatePrefs handles PATCH /me/notifications/prefs.
func (h *NotificationsHandler) UpdatePrefs(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	var req notificationsapi.UpdatePrefsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	p, err := h.uc.UpdatePrefs(c.Request().Context(), uid, req.EpisodeRelease, req.SeasonStart, req.WeeklyDigest, req.LeadTimeHours)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toAPIPrefs(p))
}

// GetTelegramLink handles GET /me/telegram/link.
func (h *NotificationsHandler) GetTelegramLink(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	url, err := h.uc.GenerateTelegramLink(c.Request().Context(), uid)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, notificationsapi.TelegramLink{Url: url})
}

// RegisterDeviceToken handles POST /me/device-token.
func (h *NotificationsHandler) RegisterDeviceToken(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	var req notificationsapi.RegisterDeviceTokenRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := h.uc.RegisterDeviceToken(c.Request().Context(), uid, req.Platform, req.Token); err != nil {
		if errors.Is(err, usecase.ErrUnknownPlatform) {
			return echo.NewHTTPError(http.StatusBadRequest, "unknown platform")
		}
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// UnregisterDeviceToken handles DELETE /me/device-token.
func (h *NotificationsHandler) UnregisterDeviceToken(c echo.Context) error {
	uid, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	if err := h.uc.UnregisterDeviceToken(c.Request().Context(), uid); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func toAPIPrefs(p *entity.NotificationPref) notificationsapi.Prefs {
	return notificationsapi.Prefs{
		EpisodeRelease: p.EpisodeRelease,
		SeasonStart:    p.SeasonStart,
		WeeklyDigest:   p.WeeklyDigest,
		Channel:        p.Channel,
		LeadTimeHours:  p.LeadTimeHours,
	}
}

func toAPINotification(n *entity.Notification) notificationsapi.NotificationItem {
	return notificationsapi.NotificationItem{
		Id:        n.ID,
		Type:      n.Type,
		Status:    n.Status,
		Read:      n.ReadAt != nil,
		Payload:   n.Payload,
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}
}
