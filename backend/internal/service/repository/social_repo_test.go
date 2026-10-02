package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/testutil"
)

func TestSocialRepository_Follows(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "follows", "users")
	userRepo := repository.NewUserRepository(gdb)
	repo := repository.NewSocialRepository(gdb)
	ctx := context.Background()

	mkUser := func(email string) *entity.User {
		u := newUser(email)
		if err := userRepo.CreateUser(ctx, u); err != nil {
			t.Fatalf("create user %s: %v", email, err)
		}
		return u
	}
	alice := mkUser("alice@s.com")
	bob := mkUser("bob@s.com")
	carol := mkUser("carol@s.com")

	// alice → bob (accepted).
	if err := repo.CreateFollow(ctx, &entity.Follow{FollowerID: alice.ID, FolloweeID: bob.ID, Status: entity.FollowAccepted}); err != nil {
		t.Fatalf("create follow: %v", err)
	}
	// Duplicate edge → ErrConflict.
	if err := repo.CreateFollow(ctx, &entity.Follow{FollowerID: alice.ID, FolloweeID: bob.ID, Status: entity.FollowAccepted}); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("dup follow: want ErrConflict, got %v", err)
	}
	// Self-follow violates the CHECK constraint.
	if err := repo.CreateFollow(ctx, &entity.Follow{FollowerID: alice.ID, FolloweeID: alice.ID, Status: entity.FollowAccepted}); err == nil {
		t.Fatal("self-follow should be rejected by CHECK")
	}

	// carol → bob (pending), plus carol → alice (pending).
	for _, fe := range []int64{bob.ID, alice.ID} {
		if err := repo.CreateFollow(ctx, &entity.Follow{FollowerID: carol.ID, FolloweeID: fe, Status: entity.FollowPending}); err != nil {
			t.Fatalf("create pending: %v", err)
		}
	}

	// Incoming to bob: only carol (pending); alice is already accepted.
	incoming, err := repo.ListIncoming(ctx, bob.ID)
	if err != nil || len(incoming) != 1 || incoming[0].ID != carol.ID {
		t.Fatalf("incoming: %v (%+v)", err, incoming)
	}

	// Accept carol → bob via status transition.
	if err := repo.SetFollowStatus(ctx, carol.ID, bob.ID, entity.FollowAccepted); err != nil {
		t.Fatalf("set status: %v", err)
	}
	f, err := repo.GetFollow(ctx, carol.ID, bob.ID)
	if err != nil || f.Status != entity.FollowAccepted {
		t.Fatalf("get follow: %v (%+v)", err, f)
	}

	// bob now has two accepted followers (alice, carol).
	if n, _ := repo.CountFollowers(ctx, bob.ID); n != 2 {
		t.Fatalf("followers count: %d", n)
	}
	followers, total, err := repo.ListFollowers(ctx, bob.ID, 10, 0)
	if err != nil || total != 2 || len(followers) != 2 {
		t.Fatalf("list followers: %v total=%d n=%d", err, total, len(followers))
	}

	// alice follows one (bob).
	if n, _ := repo.CountFollowing(ctx, alice.ID); n != 1 {
		t.Fatalf("following count: %d", n)
	}

	// AcceptAllPending for alice flips carol's pending request to accepted.
	if err := repo.AcceptAllPending(ctx, alice.ID); err != nil {
		t.Fatalf("accept all: %v", err)
	}
	if n, _ := repo.CountFollowers(ctx, alice.ID); n != 1 {
		t.Fatalf("alice followers after accept-all: %d", n)
	}

	// Unfollow removes the edge; second delete → ErrNotFound.
	if err := repo.DeleteFollow(ctx, alice.ID, bob.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.DeleteFollow(ctx, alice.ID, bob.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("second delete: want ErrNotFound, got %v", err)
	}
	if _, err := repo.GetFollow(ctx, alice.ID, bob.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("get after delete: want ErrNotFound, got %v", err)
	}
}
