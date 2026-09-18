package httpapi

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/pkg/crud"
	adminapi "github.com/deface90/defshows/backend/pkg/server/admin"
)

// AdminHandler implements the generated admin ServerInterface using the generic
// CRUD helper.
type AdminHandler struct {
	dubbing *crud.Repository[entity.DubbingStudio]
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(dubbing *crud.Repository[entity.DubbingStudio]) *AdminHandler {
	return &AdminHandler{dubbing: dubbing}
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
