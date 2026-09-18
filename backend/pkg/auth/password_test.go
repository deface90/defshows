package auth_test

import (
	"errors"
	"testing"

	"github.com/deface90/defshows/backend/pkg/auth"
)

func TestHashAndVerifyPassword(t *testing.T) {
	params := auth.DefaultArgon2Params()

	hash, err := auth.HashPassword("s3cret-pw", params)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "s3cret-pw" || len(hash) == 0 {
		t.Fatalf("unexpected hash %q", hash)
	}

	tests := []struct {
		name     string
		password string
		encoded  string
		wantErr  error
	}{
		{name: "correct password", password: "s3cret-pw", encoded: hash, wantErr: nil},
		{name: "wrong password", password: "nope", encoded: hash, wantErr: auth.ErrMismatch},
		{name: "malformed hash", password: "x", encoded: "not-a-hash", wantErr: auth.ErrInvalidHash},
		{name: "empty hash", password: "x", encoded: "", wantErr: auth.ErrInvalidHash},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.VerifyPassword(tt.password, tt.encoded)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestHashPassword_UniqueSalts(t *testing.T) {
	p := auth.DefaultArgon2Params()
	a, _ := auth.HashPassword("same", p)
	b, _ := auth.HashPassword("same", p)
	if a == b {
		t.Fatal("expected different hashes due to random salt")
	}
}
