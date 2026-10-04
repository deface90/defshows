package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/testutil"
)

func TestReportRepository(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "reports", "users")
	userRepo := repository.NewUserRepository(gdb)
	repo := repository.NewReportRepository(gdb)
	ctx := context.Background()

	mkUser := func(email string) *entity.User {
		u := newUser(email)
		if err := userRepo.CreateUser(ctx, u); err != nil {
			t.Fatalf("create user %s: %v", email, err)
		}
		return u
	}
	reporter := mkUser("reporter@r.com")
	target := mkUser("target@r.com")
	admin := mkUser("admin@r.com")

	// getByID reads a single report back via List (the repo has no Get); fails the
	// test if the id is not present.
	getByID := func(id int64) entity.Report {
		t.Helper()
		all, _, err := repo.List(ctx, "", 0, 0)
		if err != nil {
			t.Fatalf("list for getByID: %v", err)
		}
		for _, r := range all {
			if r.ID == id {
				return r
			}
		}
		t.Fatalf("getByID: report %d not found", id)
		return entity.Report{}
	}

	// Insert a report; status defaults to open.
	rep := &entity.Report{ReporterID: reporter.ID, TargetUserID: target.ID, Reason: entity.ReportSpam, Note: "buy stuff"}
	if err := repo.Create(ctx, rep); err != nil {
		t.Fatalf("create report: %v", err)
	}
	if rep.ID == 0 {
		t.Fatal("create report: want non-zero id")
	}
	if rep.Status != entity.ReportOpen {
		t.Fatalf("create report: want status open, got %q", rep.Status)
	}

	// A second report, different reason.
	rep2 := &entity.Report{ReporterID: reporter.ID, TargetUserID: target.ID, Reason: entity.ReportHarassment}
	if err := repo.Create(ctx, rep2); err != nil {
		t.Fatalf("create report 2: %v", err)
	}

	// The inserted report reads back with its fields.
	if got := getByID(rep.ID); got.Reason != entity.ReportSpam || got.Note != "buy stuff" {
		t.Fatalf("read back report: %+v", got)
	}

	// List all (empty status) → 2.
	all, total, err := repo.List(ctx, "", 10, 0)
	if err != nil || total != 2 || len(all) != 2 {
		t.Fatalf("list all: %v total=%d n=%d", err, total, len(all))
	}
	// Newest first: rep2 before rep.
	if all[0].ID != rep2.ID {
		t.Fatalf("list order: want rep2 first, got %+v", all)
	}

	// Filter by status open → 2.
	open, totalOpen, err := repo.List(ctx, string(entity.ReportOpen), 10, 0)
	if err != nil || totalOpen != 2 || len(open) != 2 {
		t.Fatalf("list open: %v total=%d n=%d", err, totalOpen, len(open))
	}
	// Filter by status resolved → 0 (none yet).
	resolved, totalResolved, err := repo.List(ctx, string(entity.ReportResolved), 10, 0)
	if err != nil || totalResolved != 0 || len(resolved) != 0 {
		t.Fatalf("list resolved: %v total=%d n=%d", err, totalResolved, len(resolved))
	}

	// Pagination: limit 1 returns one row but full total.
	page, pageTotal, err := repo.List(ctx, "", 1, 0)
	if err != nil || pageTotal != 2 || len(page) != 1 {
		t.Fatalf("list page: %v total=%d n=%d", err, pageTotal, len(page))
	}

	// Resolve rep → resolved, stamped.
	if err := repo.Resolve(ctx, rep.ID, admin.ID, entity.ReportResolved); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	gotResolved := getByID(rep.ID)
	if gotResolved.Status != entity.ReportResolved {
		t.Fatalf("resolve status: want resolved, got %q", gotResolved.Status)
	}
	if gotResolved.ResolvedAt == nil {
		t.Fatal("resolve: want resolved_at set")
	}
	if gotResolved.ResolvedBy == nil || *gotResolved.ResolvedBy != admin.ID {
		t.Fatalf("resolve: want resolved_by=%d, got %v", admin.ID, gotResolved.ResolvedBy)
	}

	// Dismiss rep2.
	if err := repo.Resolve(ctx, rep2.ID, admin.ID, entity.ReportDismissed); err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	gotDismissed := getByID(rep2.ID)
	if gotDismissed.Status != entity.ReportDismissed {
		t.Fatalf("dismiss status: want dismissed, got %q", gotDismissed.Status)
	}

	// After resolving both, status filters reflect the transitions.
	_, totalResolved2, _ := repo.List(ctx, string(entity.ReportResolved), 10, 0)
	if totalResolved2 != 1 {
		t.Fatalf("list resolved after resolve: want 1, got %d", totalResolved2)
	}
	_, totalOpen2, _ := repo.List(ctx, string(entity.ReportOpen), 10, 0)
	if totalOpen2 != 0 {
		t.Fatalf("list open after resolve: want 0, got %d", totalOpen2)
	}

	// Re-resolving an already-terminal report → ErrNotFound; the original
	// resolved_by/resolved_at are not overwritten.
	before := getByID(rep.ID)
	if err := repo.Resolve(ctx, rep.ID, target.ID, entity.ReportDismissed); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("re-resolve terminal: want ErrNotFound, got %v", err)
	}
	after := getByID(rep.ID)
	if after.Status != entity.ReportResolved || after.ResolvedBy == nil || *after.ResolvedBy != admin.ID {
		t.Fatalf("terminal report was re-stamped: before=%+v after=%+v", before, after)
	}

	// Resolve missing → ErrNotFound.
	if err := repo.Resolve(ctx, 999999, admin.ID, entity.ReportResolved); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("resolve missing: want ErrNotFound, got %v", err)
	}
}
