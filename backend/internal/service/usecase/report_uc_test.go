package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
)

// fakeReportRepo is an in-memory ReportRepo.
type fakeReportRepo struct {
	reports []entity.Report
	nextID  int64
}

func newFakeReportRepo() *fakeReportRepo { return &fakeReportRepo{nextID: 1} }

func (r *fakeReportRepo) Create(_ context.Context, rep *entity.Report) error {
	rep.ID = r.nextID
	r.nextID++
	r.reports = append(r.reports, *rep)
	return nil
}

func (r *fakeReportRepo) List(_ context.Context, status string, limit, offset int) ([]entity.Report, int64, error) {
	var matched []entity.Report
	for i := len(r.reports) - 1; i >= 0; i-- { // newest first
		if status == "" || string(r.reports[i].Status) == status {
			matched = append(matched, r.reports[i])
		}
	}
	total := int64(len(matched))
	if offset > len(matched) {
		offset = len(matched)
	}
	matched = matched[offset:]
	if limit > 0 && limit < len(matched) {
		matched = matched[:limit]
	}
	return matched, total, nil
}

func (r *fakeReportRepo) Resolve(_ context.Context, id, adminID int64, status entity.ReportStatus) error {
	for i := range r.reports {
		// Mirror the repo guard: only an open report transitions; a terminal one
		// (resolved/dismissed) is treated as not found so it is never re-stamped.
		if r.reports[i].ID == id && r.reports[i].Status == entity.ReportOpen {
			r.reports[i].Status = status
			r.reports[i].ResolvedBy = &adminID
			return nil
		}
	}
	return repository.ErrNotFound
}

type fakeReportUsers struct{ users map[int64]*entity.User }

func (u fakeReportUsers) FindUserByID(_ context.Context, id int64) (*entity.User, error) {
	if usr, ok := u.users[id]; ok {
		return usr, nil
	}
	return nil, repository.ErrNotFound
}

func newReportUC() (*usecase.ReportUsecase, *fakeReportRepo) {
	repo := newFakeReportRepo()
	users := fakeReportUsers{users: map[int64]*entity.User{
		1: {ID: 1},
		2: {ID: 2},
		3: {ID: 3},
	}}
	return usecase.NewReportUsecase(repo, users), repo
}

func TestReportUsecase_Report(t *testing.T) {
	ctx := context.Background()
	const reporter, target = int64(1), int64(2)

	// Valid report → persisted open.
	uc, _ := newReportUC()
	r, err := uc.Report(ctx, reporter, target, entity.ReportSpam, "buying followers")
	if err != nil {
		t.Fatalf("valid report: %v", err)
	}
	if r.Status != entity.ReportOpen {
		t.Fatalf("status: want open, got %s", r.Status)
	}
	if r.ReporterID != reporter || r.TargetUserID != target || r.Reason != entity.ReportSpam {
		t.Fatalf("report fields mismatch: %+v", r)
	}

	// Self-report rejected.
	if _, err := uc.Report(ctx, reporter, reporter, entity.ReportSpam, ""); !errors.Is(err, usecase.ErrSelfReport) {
		t.Fatalf("self-report: want ErrSelfReport, got %v", err)
	}

	// Invalid reason rejected.
	if _, err := uc.Report(ctx, reporter, target, entity.ReportReason("nonsense"), ""); !errors.Is(err, usecase.ErrInvalidReason) {
		t.Fatalf("invalid reason: want ErrInvalidReason, got %v", err)
	}

	// Note too long rejected.
	longNote := strings.Repeat("x", 1001)
	if _, err := uc.Report(ctx, reporter, target, entity.ReportOther, longNote); !errors.Is(err, usecase.ErrInvalidNote) {
		t.Fatalf("long note: want ErrInvalidNote, got %v", err)
	}
	// Exactly 1000 runes is accepted.
	if _, err := uc.Report(ctx, reporter, target, entity.ReportOther, strings.Repeat("x", 1000)); err != nil {
		t.Fatalf("1000-rune note: %v", err)
	}

	// Unknown target rejected.
	if _, err := uc.Report(ctx, reporter, 999, entity.ReportSpam, ""); !errors.Is(err, usecase.ErrUserNotFound) {
		t.Fatalf("unknown target: want ErrUserNotFound, got %v", err)
	}
}

func TestReportUsecase_ListReports(t *testing.T) {
	ctx := context.Background()
	uc, repo := newReportUC()

	if _, err := uc.Report(ctx, 1, 2, entity.ReportSpam, ""); err != nil {
		t.Fatalf("report 1: %v", err)
	}
	if _, err := uc.Report(ctx, 1, 3, entity.ReportHarassment, ""); err != nil {
		t.Fatalf("report 2: %v", err)
	}
	// Resolve the second one so a status filter has something to split.
	if err := uc.ResolveReport(ctx, 2, 1, entity.ReportResolved); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	// All.
	all, total, err := uc.ListReports(ctx, "", 0, 0)
	if err != nil || total != 2 || len(all) != 2 {
		t.Fatalf("list all: total=%d len=%d err=%v", total, len(all), err)
	}

	// Open only.
	open, total, err := uc.ListReports(ctx, string(entity.ReportOpen), 0, 0)
	if err != nil || total != 1 || len(open) != 1 || open[0].Status != entity.ReportOpen {
		t.Fatalf("list open: total=%d len=%d err=%v", total, len(open), err)
	}

	// Resolved only.
	resolved, total, err := uc.ListReports(ctx, string(entity.ReportResolved), 0, 0)
	if err != nil || total != 1 || len(resolved) != 1 || resolved[0].Status != entity.ReportResolved {
		t.Fatalf("list resolved: total=%d len=%d err=%v", total, len(resolved), err)
	}

	_ = repo
}

func TestReportUsecase_ResolveReport(t *testing.T) {
	ctx := context.Background()
	const admin = int64(1)
	uc, _ := newReportUC()

	if _, err := uc.Report(ctx, 1, 2, entity.ReportSpam, ""); err != nil {
		t.Fatalf("report: %v", err)
	}

	// Invalid status rejected (open is not a terminal state).
	if err := uc.ResolveReport(ctx, 1, admin, entity.ReportOpen); !errors.Is(err, usecase.ErrInvalidStatus) {
		t.Fatalf("resolve to open: want ErrInvalidStatus, got %v", err)
	}
	if err := uc.ResolveReport(ctx, 1, admin, entity.ReportStatus("bogus")); !errors.Is(err, usecase.ErrInvalidStatus) {
		t.Fatalf("resolve bogus: want ErrInvalidStatus, got %v", err)
	}

	// Resolve to dismissed stamps the admin.
	if err := uc.ResolveReport(ctx, 1, admin, entity.ReportDismissed); err != nil {
		t.Fatalf("resolve dismissed: %v", err)
	}
	dismissed, _, err := uc.ListReports(ctx, string(entity.ReportDismissed), 0, 0)
	if err != nil {
		t.Fatalf("list dismissed: %v", err)
	}
	if len(dismissed) != 1 {
		t.Fatalf("list dismissed: want 1, got %d", len(dismissed))
	}
	got := dismissed[0]
	if got.Status != entity.ReportDismissed {
		t.Fatalf("status: want dismissed, got %s", got.Status)
	}
	if got.ResolvedBy == nil || *got.ResolvedBy != admin {
		t.Fatalf("resolved_by: want %d, got %v", admin, got.ResolvedBy)
	}

	// Re-resolving an already-terminal report → ErrNotFound (no re-stamp).
	if err := uc.ResolveReport(ctx, 1, admin, entity.ReportResolved); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("re-resolve terminal: want ErrNotFound, got %v", err)
	}

	// Missing report → ErrNotFound.
	if err := uc.ResolveReport(ctx, 999, admin, entity.ReportResolved); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("resolve missing: want ErrNotFound, got %v", err)
	}
}
