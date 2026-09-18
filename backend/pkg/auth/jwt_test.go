package auth_test

import (
	"testing"
	"time"

	"github.com/deface90/defshows/backend/pkg/auth"
)

func TestJWT_GenerateParse_RoundTrip(t *testing.T) {
	m := auth.NewJWTManager("super-secret", time.Hour)

	token, err := m.Generate(42, "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != 42 || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWT_Parse_Errors(t *testing.T) {
	m := auth.NewJWTManager("secret-a", time.Hour)

	t.Run("expired", func(t *testing.T) {
		expired := auth.NewJWTManager("secret-a", -time.Minute)
		token, _ := expired.Generate(1, "user")
		if _, err := m.Parse(token); err == nil {
			t.Fatal("expected error for expired token")
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		other := auth.NewJWTManager("secret-b", time.Hour)
		token, _ := other.Generate(1, "user")
		if _, err := m.Parse(token); err == nil {
			t.Fatal("expected error for token signed with a different secret")
		}
	})

	t.Run("garbage", func(t *testing.T) {
		if _, err := m.Parse("not.a.jwt"); err == nil {
			t.Fatal("expected error for malformed token")
		}
	})
}
