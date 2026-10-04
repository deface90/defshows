package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/deface90/defshows/backend/internal/service/entity"
)

// ReportRepository handles user moderation reports.
type ReportRepository struct {
	db *gorm.DB
}

// NewReportRepository creates a ReportRepository.
func NewReportRepository(db *gorm.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// Create inserts a new report. Status defaults to open if unset (gorm sends the
// struct zero-value otherwise, bypassing the SQL default).
func (r *ReportRepository) Create(ctx context.Context, rep *entity.Report) error {
	if rep.Status == "" {
		rep.Status = entity.ReportOpen
	}
	return r.db.WithContext(ctx).Create(rep).Error
}

// List returns reports filtered by status (empty status = all), newest first,
// paginated. limit <= 0 means "no limit".
func (r *ReportRepository) List(ctx context.Context, status string, limit, offset int) ([]entity.Report, int64, error) {
	base := r.db.WithContext(ctx).Model(&entity.Report{})
	if status != "" {
		base = base.Where("status = ?", status)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := base.Order("created_at DESC, id DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	var reports []entity.Report
	if err := q.Find(&reports).Error; err != nil {
		return nil, 0, err
	}
	return reports, total, nil
}

// Resolve transitions an open report to the given terminal status, stamping
// resolved_at/resolved_by. The update is scoped to status = open so an already
// resolved/dismissed report is never re-stamped. Returns ErrNotFound if no open
// row matched (missing id or already terminal).
func (r *ReportRepository) Resolve(ctx context.Context, id, adminID int64, status entity.ReportStatus) error {
	res := r.db.WithContext(ctx).Model(&entity.Report{}).
		Where("id = ? AND status = ?", id, entity.ReportOpen).
		Updates(map[string]any{
			"status":      status,
			"resolved_at": gorm.Expr("now()"),
			"resolved_by": adminID,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
