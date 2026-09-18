package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/pkg/auth"
)

// fakeRepo is an in-memory UserRepo for usecase tests.
type fakeRepo struct {
	users      map[int64]*entity.User
	emailIndex map[string]int64
	refresh    map[string]*entity.RefreshToken
	identities []*entity.UserIdentity
	handoffs   map[string]int64
	nextUID    int64
	nextRTID   int64
}

func newFake() *fakeRepo {
	return &fakeRepo{
		users:      map[int64]*entity.User{},
		emailIndex: map[string]int64{},
		refresh:    map[string]*entity.RefreshToken{},
		handoffs:   map[string]int64{},
	}
}

func (f *fakeRepo) CreateUser(_ context.Context, u *entity.User) error {
	if u.Email != nil {
		if _, ok := f.emailIndex[*u.Email]; ok {
			return repository.ErrConflict
		}
	}
	f.nextUID++
	u.ID = f.nextUID
	f.users[u.ID] = u
	if u.Email != nil {
		f.emailIndex[*u.Email] = u.ID
	}
	return nil
}

func (f *fakeRepo) FindUserByEmail(_ context.Context, email string) (*entity.User, error) {
	id, ok := f.emailIndex[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return f.users[id], nil
}

func (f *fakeRepo) FindUserByID(_ context.Context, id int64) (*entity.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (f *fakeRepo) UpdateTimezone(_ context.Context, id int64, tz string) error {
	if u, ok := f.users[id]; ok {
		u.Timezone = tz
	}
	return nil
}

func (f *fakeRepo) ListUsers(_ context.Context) ([]entity.User, error) {
	out := make([]entity.User, 0, len(f.users))
	for _, u := range f.users {
		out = append(out, *u)
	}
	return out, nil
}

func (f *fakeRepo) SetPublic(_ context.Context, id int64, public bool) error {
	if u, ok := f.users[id]; ok {
		u.IsPublic = public
	}
	return nil
}

func (f *fakeRepo) FindIdentity(_ context.Context, provider, providerUserID string) (*entity.UserIdentity, error) {
	for _, id := range f.identities {
		if id.Provider == provider && id.ProviderUserID == providerUserID {
			return id, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeRepo) CreateIdentity(_ context.Context, id *entity.UserIdentity) error {
	f.identities = append(f.identities, id)
	return nil
}

func (f *fakeRepo) CreateOAuthHandoff(_ context.Context, code string, userID int64, _ time.Time) error {
	f.handoffs[code] = userID
	return nil
}

func (f *fakeRepo) ConsumeOAuthHandoff(_ context.Context, code string) (int64, error) {
	uid, ok := f.handoffs[code]
	if !ok {
		return 0, repository.ErrNotFound
	}
	delete(f.handoffs, code)
	return uid, nil
}

func (f *fakeRepo) SaveRefreshToken(_ context.Context, rt *entity.RefreshToken) error {
	f.nextRTID++
	rt.ID = f.nextRTID
	f.refresh[rt.TokenHash] = rt
	return nil
}

func (f *fakeRepo) FindRefreshByHash(_ context.Context, hash string) (*entity.RefreshToken, error) {
	rt, ok := f.refresh[hash]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return rt, nil
}

func (f *fakeRepo) RevokeRefreshToken(_ context.Context, id int64) error {
	for _, rt := range f.refresh {
		if rt.ID == id && rt.RevokedAt == nil {
			now := time.Now()
			rt.RevokedAt = &now
		}
	}
	return nil
}

func (f *fakeRepo) RevokeFamily(_ context.Context, family string) error {
	for _, rt := range f.refresh {
		if rt.FamilyID == family && rt.RevokedAt == nil {
			now := time.Now()
			rt.RevokedAt = &now
		}
	}
	return nil
}

func (f *fakeRepo) activeTokens() int {
	n := 0
	for _, rt := range f.refresh {
		if rt.RevokedAt == nil {
			n++
		}
	}
	return n
}

// backdateRevoked ages every revoked token so it falls outside the reuse grace
// window (simulates a genuine, later replay rather than a concurrent race).
func (f *fakeRepo) backdateRevoked(age time.Duration) {
	for _, rt := range f.refresh {
		if rt.RevokedAt != nil {
			t := time.Now().Add(-age)
			rt.RevokedAt = &t
		}
	}
}

func newUC(repo usecase.UserRepo) *usecase.AuthUsecase {
	return usecase.NewAuthUsecase(repo, auth.NewJWTManager("test-secret", time.Hour), time.Hour)
}

func TestAuth_SeedAdmin(t *testing.T) {
	repo := newFake()
	uc := newUC(repo)
	ctx := context.Background()

	// Blank creds → no-op.
	if seeded, err := uc.SeedAdmin(ctx, "", ""); err != nil || seeded {
		t.Fatalf("blank seed should be a no-op, got seeded=%v err=%v", seeded, err)
	}

	// First seed creates an admin that can log in.
	seeded, err := uc.SeedAdmin(ctx, "admin@x.io", "s3cret-pw")
	if err != nil || !seeded {
		t.Fatalf("first seed should create, got seeded=%v err=%v", seeded, err)
	}
	u, err := repo.FindUserByEmail(ctx, "admin@x.io")
	if err != nil || u.Role != entity.RoleAdmin {
		t.Fatalf("expected admin user, got %+v err=%v", u, err)
	}
	if _, _, err := uc.Login(ctx, "admin@x.io", "s3cret-pw", "agent"); err != nil {
		t.Fatalf("seeded admin login: %v", err)
	}

	// Second seed is idempotent (existing email untouched).
	if seeded, err := uc.SeedAdmin(ctx, "admin@x.io", "different"); err != nil || seeded {
		t.Fatalf("re-seed should be a no-op, got seeded=%v err=%v", seeded, err)
	}
}

func TestAuth_RegisterAndLogin(t *testing.T) {
	repo := newFake()
	uc := newUC(repo)
	ctx := context.Background()

	u, pair, err := uc.Register(ctx, "a@b.c", "pw12345", "agent")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if u.ID == 0 || pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("unexpected register result: %+v / %+v", u, pair)
	}

	// Duplicate registration.
	if _, _, err := uc.Register(ctx, "a@b.c", "pw12345", "agent"); !errors.Is(err, usecase.ErrEmailTaken) {
		t.Fatalf("want ErrEmailTaken, got %v", err)
	}

	// Login success.
	if _, _, err := uc.Login(ctx, "a@b.c", "pw12345", "agent"); err != nil {
		t.Fatalf("login: %v", err)
	}

	tests := []struct {
		name, email, password string
	}{
		{"wrong password", "a@b.c", "wrong"},
		{"unknown email", "no@b.c", "pw12345"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := uc.Login(ctx, tt.email, tt.password, "agent"); !errors.Is(err, usecase.ErrInvalidCredentials) {
				t.Fatalf("want ErrInvalidCredentials, got %v", err)
			}
		})
	}
}

func TestAuth_Login_SocialOnly(t *testing.T) {
	repo := newFake()
	uc := newUC(repo)
	ctx := context.Background()

	email := "social@b.c"
	_ = repo.CreateUser(ctx, &entity.User{Email: &email, Role: entity.RoleUser}) // no password hash

	if _, _, err := uc.Login(ctx, email, "anything", "agent"); !errors.Is(err, usecase.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials for social-only, got %v", err)
	}
}

func TestAuth_Refresh_RotationAndReuse(t *testing.T) {
	repo := newFake()
	uc := newUC(repo)
	ctx := context.Background()

	_, pair, err := uc.Register(ctx, "r@b.c", "pw12345", "agent")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Rotate once: old token revoked, exactly one active token remains.
	newPair, err := uc.Refresh(ctx, pair.RefreshToken, "agent")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Fatal("expected a rotated refresh token")
	}
	if repo.activeTokens() != 1 {
		t.Fatalf("want 1 active token after rotation, got %d", repo.activeTokens())
	}

	// Benign race: the just-revoked token replayed within the grace window is
	// reissued in the same family (concurrent tabs / reload) instead of logging
	// the user out.
	racePair, err := uc.Refresh(ctx, pair.RefreshToken, "agent")
	if err != nil {
		t.Fatalf("want grace reissue on immediate reuse, got %v", err)
	}
	if racePair.RefreshToken == "" {
		t.Fatal("expected a reissued refresh token within grace window")
	}
	if repo.activeTokens() != 2 {
		t.Fatalf("want 2 active tokens after grace reissue, got %d", repo.activeTokens())
	}

	// Genuine reuse: the same revoked token replayed past the grace window kills
	// the whole family.
	repo.backdateRevoked(time.Hour)
	if _, err := uc.Refresh(ctx, pair.RefreshToken, "agent"); !errors.Is(err, usecase.ErrInvalidRefresh) {
		t.Fatalf("want ErrInvalidRefresh on reuse past grace, got %v", err)
	}
	if repo.activeTokens() != 0 {
		t.Fatalf("want 0 active tokens after reuse detection, got %d", repo.activeTokens())
	}
}

func TestAuth_Refresh_Errors(t *testing.T) {
	repo := newFake()
	uc := newUC(repo)
	ctx := context.Background()

	if _, err := uc.Refresh(ctx, "unknown-token", "agent"); !errors.Is(err, usecase.ErrInvalidRefresh) {
		t.Fatalf("want ErrInvalidRefresh for unknown, got %v", err)
	}

	_, pair, _ := uc.Register(ctx, "e@b.c", "pw12345", "agent")
	// Expire the stored token.
	for _, rt := range repo.refresh {
		rt.ExpiresAt = time.Now().Add(-time.Minute)
	}
	if _, err := uc.Refresh(ctx, pair.RefreshToken, "agent"); !errors.Is(err, usecase.ErrInvalidRefresh) {
		t.Fatalf("want ErrInvalidRefresh for expired, got %v", err)
	}
}
