package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/pkg/crud"
	adminapi "github.com/deface90/defshows/backend/pkg/server/admin"
)

// AdminUserLookup resolves users for report reporter/target enrichment.
type AdminUserLookup interface {
	FindUserByID(ctx context.Context, id int64) (*entity.User, error)
}

// AdminHandler implements the generated admin ServerInterface using the generic
// CRUD helper plus moderation-report review.
type AdminHandler struct {
	dubbing *crud.Repository[entity.DubbingStudio]
	report  *usecase.ReportUsecase
	users   AdminUserLookup
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(dubbing *crud.Repository[entity.DubbingStudio], reportUC *usecase.ReportUsecase, users AdminUserLookup) *AdminHandler {
	return &AdminHandler{dubbing: dubbing, report: reportUC, users: users}
}

var _ adminapi.ServerInterface = (*AdminHandler)(nil)

// ListDubbingStudios handles GET /admin/dubbing-studios.
func (h *AdminHandler) ListDubbingStudios(c echo.Context) error {
	items, err := h.dubbing.List(c.Request().Context(), 200, 0)
	if err != nil {
		return err
	}
	out := make([]adminapi.DubbingStudio, 0, len(items))
	for i := range items {
		out = append(out, toAPIDubbing(&items[i]))
	}
	return c.JSON(http.StatusOK, adminapi.DubbingStudioList{Studios: out})
}

// CreateDubbingStudio handles POST /admin/dubbing-studios.
func (h *AdminHandler) CreateDubbingStudio(c echo.Context) error {
	var req adminapi.DubbingStudioRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	ds := &entity.DubbingStudio{Name: req.Name, SiteURL: strval(req.SiteUrl), Active: boolval(req.Active, true)}
	if err := h.dubbing.Create(c.Request().Context(), ds); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, toAPIDubbing(ds))
}

// UpdateDubbingStudio handles PUT /admin/dubbing-studios/{id}.
func (h *AdminHandler) UpdateDubbingStudio(c echo.Context, id int64) error {
	var req adminapi.DubbingStudioRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	ds, err := h.dubbing.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, crud.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}
		return err
	}
	ds.Name = req.Name
	ds.SiteURL = strval(req.SiteUrl)
	ds.Active = boolval(req.Active, ds.Active)
	if err := h.dubbing.Save(c.Request().Context(), ds); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toAPIDubbing(ds))
}

// DeleteDubbingStudio handles DELETE /admin/dubbing-studios/{id}.
func (h *AdminHandler) DeleteDubbingStudio(c echo.Context, id int64) error {
	if err := h.dubbing.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, crud.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// ListReports handles GET /admin/reports, optionally filtered by status,
// enriching each report with reporter/target display-name summaries.
func (h *AdminHandler) ListReports(c echo.Context, params adminapi.ListReportsParams) error {
	ctx := c.Request().Context()
	status := ""
	if params.Status != nil {
		status = string(*params.Status)
	}
	p := pageParams(params.Page, params.PageSize)
	reports, total, err := h.report.ListReports(ctx, status, p.limit, p.offset)
	if err != nil {
		return err
	}
	// Cache user lookups across the page so a prolific reporter is fetched once.
	cache := map[int64]adminapi.ReportUserSummary{}
	summary := func(id int64) adminapi.ReportUserSummary {
		if s, ok := cache[id]; ok {
			return s
		}
		s := adminapi.ReportUserSummary{Id: id, DisplayName: fmt.Sprintf("Пользователь #%d", id)}
		if u, err := h.users.FindUserByID(ctx, id); err == nil {
			s.DisplayName = displayNameOf(u)
		}
		cache[id] = s
		return s
	}
	out := make([]adminapi.Report, 0, len(reports))
	for i := range reports {
		r := reports[i]
		item := adminapi.Report{
			Id:        r.ID,
			Reporter:  summary(r.ReporterID),
			Target:    summary(r.TargetUserID),
			Reason:    adminapi.ReportReason(r.Reason),
			Note:      r.Note,
			Status:    adminapi.ReportStatus(r.Status),
			CreatedAt: r.CreatedAt,
		}
		if r.ResolvedAt != nil {
			item.ResolvedAt = r.ResolvedAt
		}
		out = append(out, item)
	}
	return c.JSON(http.StatusOK, adminapi.ReportList{Reports: out, Total: total})
}

// ResolveReport handles POST /admin/reports/{id}/resolve.
func (h *AdminHandler) ResolveReport(c echo.Context, id int64) error {
	var req adminapi.ResolveReportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	adminID, ok := userID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
	}
	err := h.report.ResolveReport(c.Request().Context(), id, adminID, entity.ReportStatus(req.Status))
	switch {
	case errors.Is(err, usecase.ErrInvalidStatus):
		return echo.NewHTTPError(http.StatusBadRequest, "invalid status")
	case errors.Is(err, repository.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	case err != nil:
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func toAPIDubbing(d *entity.DubbingStudio) adminapi.DubbingStudio {
	site := d.SiteURL
	return adminapi.DubbingStudio{Id: d.ID, Name: d.Name, SiteUrl: &site, Active: d.Active}
}

func strval(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func boolval(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}
