package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/deface90/defshows/backend/internal/service/usecase"
)

func TestAuth_LoginWithOAuth(t *testing.T) {
	repo := newFake()
	uc := newUC(repo)
	ctx := context.Background()

	// New identity → creates a user.
	u1, err := uc.LoginWithOAuth(ctx, "google", "g-1", "new@ex.com", "New User")
	if err != nil {
		t.Fatalf("oauth login: %v", err)
	}
	if u1.ID == 0 || u1.Email == nil || *u1.Email != "new@ex.com" {
		t.Fatalf("unexpected user: %+v", u1)
	}

	// Same identity again → same user.
	u2, err := uc.LoginWithOAuth(ctx, "google", "g-1", "new@ex.com", "New User")
	if err != nil {
		t.Fatalf("oauth login 2: %v", err)
	}
	if u2.ID != u1.ID {
		t.Fatalf("want same user, got %d vs %d", u2.ID, u1.ID)
	}

	// Different provider identity but same email → links to existing user.
	u3, err := uc.LoginWithOAuth(ctx, "yandex", "y-9", "new@ex.com", "New User")
	if err != nil {
		t.Fatalf("oauth login 3: %v", err)
	}
	if u3.ID != u1.ID {
		t.Fatalf("expected link to existing user %d, got %d", u1.ID, u3.ID)
	}
}

func TestAuth_OAuthHandoff(t *testing.T) {
	repo := newFake()
	uc := newUC(repo)
	ctx := context.Background()

	user, err := uc.LoginWithOAuth(ctx, "google", "g-2", "h@ex.com", "H")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	code, err := uc.CreateOAuthHandoff(ctx, user.ID)
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}

	got, pair, err := uc.ExchangeOAuthHandoff(ctx, code, "agent")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if got.ID != user.ID || pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("unexpected exchange: %+v / %+v", got, pair)
	}

	// One-time: reuse fails.
	if _, _, err := uc.ExchangeOAuthHandoff(ctx, code, "agent"); !errors.Is(err, usecase.ErrInvalidHandoff) {
		t.Fatalf("want ErrInvalidHandoff on reuse, got %v", err)
	}
}
