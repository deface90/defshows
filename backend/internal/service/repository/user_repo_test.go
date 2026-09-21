package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/testutil"
)

func strptr(s string) *string { return &s }

func newUser(email string) *entity.User {
	return &entity.User{
		Email:        strptr(email),
		PasswordHash: strptr("hash"),
		DisplayName:  "Test",
		Role:         entity.RoleUser,
		Timezone:     "UTC",
	}
}

func TestUserRepository_Users(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users")
	repo := repository.NewUserRepository(gdb)
	ctx := context.Background()

	u := newUser("alice@example.com")
	if err := repo.CreateUser(ctx, u); err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("expected generated id")
	}

	got, err := repo.FindUserByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("find by email: %v", err)
	}
	if got.ID != u.ID {
		t.Fatalf("want id %d, got %d", u.ID, got.ID)
	}

	byID, err := repo.FindUserByID(ctx, u.ID)
	if err != nil || byID.ID != u.ID {
		t.Fatalf("find by id: %v (%+v)", err, byID)
	}

	if _, err := repo.FindUserByEmail(ctx, "nobody@example.com"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	// Duplicate email → ErrConflict.
	if err := repo.CreateUser(ctx, newUser("alice@example.com")); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestUserRepository_RefreshTokens(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users")
	repo := repository.NewUserRepository(gdb)
	ctx := context.Background()

	u := newUser("bob@example.com")
	if err := repo.CreateUser(ctx, u); err != nil {
		t.Fatalf("create user: %v", err)
	}

	family := uuid.NewString()
	rt := &entity.RefreshToken{
		UserID:    u.ID,
		TokenHash: "hash-1",
		FamilyID:  family,
		ExpiresAt: time.Now().Add(time.Hour),
		UserAgent: "test",
	}
	if err := repo.SaveRefreshToken(ctx, rt); err != nil {
		t.Fatalf("save refresh: %v", err)
	}

	found, err := repo.FindRefreshByHash(ctx, "hash-1")
	if err != nil {
		t.Fatalf("find refresh: %v", err)
	}
	if found.RevokedAt != nil {
		t.Fatal("new token should not be revoked")
	}

	if err := repo.RevokeRefreshToken(ctx, found.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	found, _ = repo.FindRefreshByHash(ctx, "hash-1")
	if found.RevokedAt == nil {
		t.Fatal("token should be revoked")
	}

	// Second token in same family, then revoke family.
	rt2 := &entity.RefreshToken{
		UserID:    u.ID,
		TokenHash: "hash-2",
		FamilyID:  family,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := repo.SaveRefreshToken(ctx, rt2); err != nil {
		t.Fatalf("save refresh 2: %v", err)
	}
	if err := repo.RevokeFamily(ctx, family); err != nil {
		t.Fatalf("revoke family: %v", err)
	}
	found2, _ := repo.FindRefreshByHash(ctx, "hash-2")
	if found2.RevokedAt == nil {
		t.Fatal("family revoke should revoke hash-2")
	}

	if _, err := repo.FindRefreshByHash(ctx, "missing"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestUserRepository_DirectoryPaginationAndLiteralSearch(t *testing.T) {
	db := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, db, "users")
	repo := repository.NewUserRepository(db)
	ctx := t.Context()
	for i, name := range []string{"Alice", "Alice", "Bob_", "Bob%", "Charlie"} {
		u := newUser(string(rune('a'+i)) + "@example.com")
		u.DisplayName = name
		if err := repo.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	first, total, err := repo.ListUsers(ctx, "alice", 1, 0)
	if err != nil || total != 2 || len(first) != 1 {
		t.Fatalf("first page: %+v %d %v", first, total, err)
	}
	second, total, err := repo.ListUsers(ctx, "ALICE", 1, 1)
	if err != nil || total != 2 || len(second) != 1 || second[0].ID <= first[0].ID {
		t.Fatalf("unstable pages: %+v %d %v", second, total, err)
	}
	for _, tc := range []struct{ q, name string }{{"%", "Bob%"}, {"_", "Bob_"}} {
		users, total, err := repo.ListUsers(ctx, tc.q, 20, 0)
		if err != nil || total != 1 || len(users) != 1 || users[0].DisplayName != tc.name {
			t.Fatalf("literal search %q: %+v %d %v", tc.q, users, total, err)
		}
	}
	users, total, err := repo.ListUsers(ctx, "", 2, 10)
	if err != nil || total != 5 || len(users) != 0 {
		t.Fatalf("out of range: %+v %d %v", users, total, err)
	}
}
