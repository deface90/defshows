package usecase

import (
	"context"
	"errors"
	"unicode/utf8"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
)

// Report usecase errors.
var (
	ErrSelfReport    = errors.New("usecase: cannot report yourself")
	ErrInvalidReason = errors.New("usecase: invalid report reason")
	ErrInvalidNote   = errors.New("usecase: report note too long")
	ErrInvalidStatus = errors.New("usecase: invalid report status")
)

// reportNoteMaxLen caps the free-text note length (in runes).
const reportNoteMaxLen = 1000

// ReportRepo is the report-storage dependency of ReportUsecase.
type ReportRepo interface {
	Create(ctx context.Context, r *entity.Report) error
	List(ctx context.Context, status string, limit, offset int) ([]entity.Report, int64, error)
	Resolve(ctx context.Context, id, adminID int64, status entity.ReportStatus) error
}

// ReportUserRepo looks up users for report existence checks (mirrors SocialUserRepo).
type ReportUserRepo interface {
	FindUserByID(ctx context.Context, id int64) (*entity.User, error)
}

// ReportUsecase implements user-filed moderation reports and admin review.
type ReportUsecase struct {
	repo  ReportRepo
	users ReportUserRepo
}

// NewReportUsecase creates a ReportUsecase.
func NewReportUsecase(repo ReportRepo, users ReportUserRepo) *ReportUsecase {
	return &ReportUsecase{repo: repo, users: users}
}

// validReportReason reports whether reason is one of the allowed values.
func validReportReason(reason entity.ReportReason) bool {
	switch reason {
	case entity.ReportSpam, entity.ReportHarassment, entity.ReportInappropriate, entity.ReportOther:
		return true
	default:
		return false
	}
}

// Report files a moderation report from reporterID against targetID. It validates the
// reason, caps the note length, rejects self-reports, and verifies the target exists;
// the report is persisted with status "open" for admin review.
func (uc *ReportUsecase) Report(ctx context.Context, reporterID, targetID int64, reason entity.ReportReason, note string) (*entity.Report, error) {
	if reporterID == targetID {
		return nil, ErrSelfReport
	}
	if !validReportReason(reason) {
		return nil, ErrInvalidReason
	}
	if utf8.RuneCountInString(note) > reportNoteMaxLen {
		return nil, ErrInvalidNote
	}
	if _, err := uc.users.FindUserByID(ctx, targetID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	r := &entity.Report{
		ReporterID:   reporterID,
		TargetUserID: targetID,
		Reason:       reason,
		Note:         note,
		Status:       entity.ReportOpen,
	}
	if err := uc.repo.Create(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// ListReports returns reports filtered by status (empty = all), paginated.
func (uc *ReportUsecase) ListReports(ctx context.Context, status string, limit, offset int) ([]entity.Report, int64, error) {
	return uc.repo.List(ctx, status, limit, offset)
}

// ResolveReport transitions a report to a terminal status (resolved or dismissed),
// stamping the acting admin. Returns ErrInvalidStatus for any other status.
func (uc *ReportUsecase) ResolveReport(ctx context.Context, id, adminID int64, status entity.ReportStatus) error {
	if status != entity.ReportResolved && status != entity.ReportDismissed {
		return ErrInvalidStatus
	}
	return uc.repo.Resolve(ctx, id, adminID, status)
}
