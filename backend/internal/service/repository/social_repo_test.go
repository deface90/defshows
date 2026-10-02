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

func TestSocialRepository_Blocks(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "blocks", "follows", "users")
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
	alice := mkUser("alice@b.com")
	bob := mkUser("bob@b.com")
	carol := mkUser("carol@b.com")

	// alice blocks bob.
	if err := repo.CreateBlock(ctx, &entity.Block{BlockerID: alice.ID, BlockedID: bob.ID}); err != nil {
		t.Fatalf("create block: %v", err)
	}
	// Duplicate (blocker, blocked) → ErrConflict.
	if err := repo.CreateBlock(ctx, &entity.Block{BlockerID: alice.ID, BlockedID: bob.ID}); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("dup block: want ErrConflict, got %v", err)
	}
	// Self-block violates the CHECK constraint.
	if err := repo.CreateBlock(ctx, &entity.Block{BlockerID: alice.ID, BlockedID: alice.ID}); err == nil {
		t.Fatal("self-block should be rejected by CHECK")
	}

	// IsBlockedEither true in both directions for alice/bob.
	if blocked, err := repo.IsBlockedEither(ctx, alice.ID, bob.ID); err != nil || !blocked {
		t.Fatalf("IsBlockedEither alice,bob: want true, got %v (%v)", blocked, err)
	}
	if blocked, err := repo.IsBlockedEither(ctx, bob.ID, alice.ID); err != nil || !blocked {
		t.Fatalf("IsBlockedEither bob,alice: want true, got %v (%v)", blocked, err)
	}
	// No block between alice and carol.
	if blocked, err := repo.IsBlockedEither(ctx, alice.ID, carol.ID); err != nil || blocked {
		t.Fatalf("IsBlockedEither alice,carol: want false, got %v (%v)", blocked, err)
	}

	// carol blocks alice → BlockedIDsEither(alice) = {bob (outgoing), carol (incoming)}.
	if err := repo.CreateBlock(ctx, &entity.Block{BlockerID: carol.ID, BlockedID: alice.ID}); err != nil {
		t.Fatalf("carol block alice: %v", err)
	}
	ids, err := repo.BlockedIDsEither(ctx, alice.ID)
	if err != nil {
		t.Fatalf("BlockedIDsEither: %v", err)
	}
	gotIDs := map[int64]bool{}
	for _, id := range ids {
		gotIDs[id] = true
	}
	if len(ids) != 2 || !gotIDs[bob.ID] || !gotIDs[carol.ID] {
		t.Fatalf("BlockedIDsEither(alice): want {bob,carol}, got %v", ids)
	}

	// ListBlocked(alice) returns only bob (users alice blocked, not who blocked alice).
	blockedUsers, total, err := repo.ListBlocked(ctx, alice.ID, 10, 0)
	if err != nil || total != 1 || len(blockedUsers) != 1 || blockedUsers[0].ID != bob.ID {
		t.Fatalf("ListBlocked(alice): want {bob} total=1, got total=%d users=%+v err=%v", total, blockedUsers, err)
	}
	// Pagination: limit 0 still returns, offset past end returns empty page but full total.
	_, total2, err := repo.ListBlocked(ctx, alice.ID, 10, 5)
	if err != nil || total2 != 1 {
		t.Fatalf("ListBlocked pagination offset: total=%d err=%v", total2, err)
	}

	// DeleteBlock removes the directed row; second delete → ErrNotFound.
	if err := repo.DeleteBlock(ctx, alice.ID, bob.ID); err != nil {
		t.Fatalf("delete block: %v", err)
	}
	if err := repo.DeleteBlock(ctx, alice.ID, bob.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("second delete block: want ErrNotFound, got %v", err)
	}
	// Reverse direction (carol→alice) still blocks the alice/carol pair.
	if blocked, err := repo.IsBlockedEither(ctx, alice.ID, carol.ID); err != nil || !blocked {
		t.Fatalf("IsBlockedEither alice,carol after unblock bob: want true, got %v (%v)", blocked, err)
	}
}

func TestSocialRepository_DeleteFollowEither(t *testing.T) {
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
	alice := mkUser("alice@dfe.com")
	bob := mkUser("bob@dfe.com")

	// Both directions exist.
	if err := repo.CreateFollow(ctx, &entity.Follow{FollowerID: alice.ID, FolloweeID: bob.ID, Status: entity.FollowAccepted}); err != nil {
		t.Fatalf("create a→b: %v", err)
	}
	if err := repo.CreateFollow(ctx, &entity.Follow{FollowerID: bob.ID, FolloweeID: alice.ID, Status: entity.FollowPending}); err != nil {
		t.Fatalf("create b→a: %v", err)
	}

	// DeleteFollowEither removes both edges in one call.
	if err := repo.DeleteFollowEither(ctx, alice.ID, bob.ID); err != nil {
		t.Fatalf("DeleteFollowEither: %v", err)
	}
	if _, err := repo.GetFollow(ctx, alice.ID, bob.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("a→b after DeleteFollowEither: want ErrNotFound, got %v", err)
	}
	if _, err := repo.GetFollow(ctx, bob.ID, alice.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("b→a after DeleteFollowEither: want ErrNotFound, got %v", err)
	}

	// Idempotent: calling again with no edges is not an error.
	if err := repo.DeleteFollowEither(ctx, alice.ID, bob.ID); err != nil {
		t.Fatalf("DeleteFollowEither idempotent: %v", err)
	}
}
