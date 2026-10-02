package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
)

// fakeSocialRepo is an in-memory SocialRepo keyed by (follower, followee).
type fakeSocialRepo struct {
	edges map[[2]int64]*entity.Follow
}

func newFakeSocialRepo() *fakeSocialRepo {
	return &fakeSocialRepo{edges: map[[2]int64]*entity.Follow{}}
}

func (r *fakeSocialRepo) CreateFollow(_ context.Context, f *entity.Follow) error {
	k := [2]int64{f.FollowerID, f.FolloweeID}
	if _, ok := r.edges[k]; ok {
		return repository.ErrConflict
	}
	cp := *f
	r.edges[k] = &cp
	return nil
}
func (r *fakeSocialRepo) GetFollow(_ context.Context, follower, followee int64) (*entity.Follow, error) {
	if f, ok := r.edges[[2]int64{follower, followee}]; ok {
		cp := *f
		return &cp, nil
	}
	return nil, repository.ErrNotFound
}
func (r *fakeSocialRepo) DeleteFollow(_ context.Context, follower, followee int64) error {
	k := [2]int64{follower, followee}
	if _, ok := r.edges[k]; !ok {
		return repository.ErrNotFound
	}
	delete(r.edges, k)
	return nil
}
func (r *fakeSocialRepo) SetFollowStatus(_ context.Context, follower, followee int64, s entity.FollowStatus) error {
	k := [2]int64{follower, followee}
	f, ok := r.edges[k]
	if !ok {
		return repository.ErrNotFound
	}
	f.Status = s
	return nil
}
func (r *fakeSocialRepo) AcceptAllPending(_ context.Context, followee int64) error {
	for _, f := range r.edges {
		if f.FolloweeID == followee && f.Status == entity.FollowPending {
			f.Status = entity.FollowAccepted
		}
	}
	return nil
}
func (r *fakeSocialRepo) ListFollowers(_ context.Context, userID int64, _, _ int) ([]entity.User, int64, error) {
	var out []entity.User
	for _, f := range r.edges {
		if f.FolloweeID == userID && f.Status == entity.FollowAccepted {
			out = append(out, entity.User{ID: f.FollowerID})
		}
	}
	return out, int64(len(out)), nil
}
func (r *fakeSocialRepo) ListFollowing(_ context.Context, userID int64, _, _ int) ([]entity.User, int64, error) {
	var out []entity.User
	for _, f := range r.edges {
		if f.FollowerID == userID && f.Status == entity.FollowAccepted {
			out = append(out, entity.User{ID: f.FolloweeID})
		}
	}
	return out, int64(len(out)), nil
}
func (r *fakeSocialRepo) ListIncoming(_ context.Context, userID int64) ([]entity.User, error) {
	var out []entity.User
	for _, f := range r.edges {
		if f.FolloweeID == userID && f.Status == entity.FollowPending {
			out = append(out, entity.User{ID: f.FollowerID})
		}
	}
	return out, nil
}
func (r *fakeSocialRepo) CountFollowers(_ context.Context, userID int64) (int64, error) {
	var n int64
	for _, f := range r.edges {
		if f.FolloweeID == userID && f.Status == entity.FollowAccepted {
			n++
		}
	}
	return n, nil
}
func (r *fakeSocialRepo) CountFollowing(_ context.Context, userID int64) (int64, error) {
	var n int64
	for _, f := range r.edges {
		if f.FollowerID == userID && f.Status == entity.FollowAccepted {
			n++
		}
	}
	return n, nil
}

type fakeSocialUsers struct{ users map[int64]*entity.User }

func (u fakeSocialUsers) FindUserByID(_ context.Context, id int64) (*entity.User, error) {
	if usr, ok := u.users[id]; ok {
		return usr, nil
	}
	return nil, repository.ErrNotFound
}

// fakeFollowNotifier records emitted in-app notifications and honours dedupe + delete.
type fakeFollowNotifier struct{ created []entity.Notification }

func (f *fakeFollowNotifier) CreateNotificationIfAbsent(_ context.Context, n *entity.Notification) (bool, error) {
	for i := range f.created {
		if f.created[i].DedupeKey == n.DedupeKey {
			return false, nil
		}
	}
	f.created = append(f.created, *n)
	return true, nil
}

func (f *fakeFollowNotifier) DeleteByDedupeKeys(_ context.Context, keys ...string) error {
	del := map[string]bool{}
	for _, k := range keys {
		del[k] = true
	}
	out := f.created[:0]
	for _, n := range f.created {
		if !del[n.DedupeKey] {
			out = append(out, n)
		}
	}
	f.created = out
	return nil
}

func newSocialUC(public, private int64) (*usecase.SocialUsecase, *fakeFollowNotifier) {
	users := fakeSocialUsers{users: map[int64]*entity.User{
		public:  {ID: public, IsPublic: true},
		private: {ID: private, IsPublic: false},
	}}
	notifier := &fakeFollowNotifier{}
	return usecase.NewSocialUsecase(newFakeSocialRepo(), users, notifier), notifier
}

func TestSocialUsecase_Follow(t *testing.T) {
	ctx := context.Background()
	const viewer, pub, priv = int64(1), int64(2), int64(3)
	uc, _ := newSocialUC(pub, priv)
	// register the viewer too so CanViewProfile(target=viewer) works if needed
	_ = viewer

	// Self-follow rejected.
	if _, err := uc.Follow(ctx, pub, pub); !errors.Is(err, usecase.ErrSelfFollow) {
		t.Fatalf("self-follow: want ErrSelfFollow, got %v", err)
	}
	// Unknown target rejected.
	if _, err := uc.Follow(ctx, pub, 999); !errors.Is(err, usecase.ErrUserNotFound) {
		t.Fatalf("unknown target: want ErrUserNotFound, got %v", err)
	}
	// Following a public user → accepted immediately.
	f, err := uc.Follow(ctx, priv, pub)
	if err != nil || f.Status != entity.FollowAccepted {
		t.Fatalf("follow public: %v (%+v)", err, f)
	}
	// Following a private user → pending.
	f, err = uc.Follow(ctx, pub, priv)
	if err != nil || f.Status != entity.FollowPending {
		t.Fatalf("follow private: %v (%+v)", err, f)
	}
	// Repeat follow is idempotent (returns existing pending edge).
	again, err := uc.Follow(ctx, pub, priv)
	if err != nil || again.Status != entity.FollowPending {
		t.Fatalf("repeat follow: %v (%+v)", err, again)
	}

	// Approve the pending pub→priv request.
	if err := uc.Approve(ctx, priv, pub); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if state, _ := uc.FollowState(ctx, pub, priv); state != "accepted" {
		t.Fatalf("state after approve: %s", state)
	}

	// Unfollow is idempotent.
	if err := uc.Unfollow(ctx, pub, priv); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	if err := uc.Unfollow(ctx, pub, priv); err != nil {
		t.Fatalf("second unfollow should be no-op: %v", err)
	}
	if state, _ := uc.FollowState(ctx, pub, priv); state != "none" {
		t.Fatalf("state after unfollow: %s", state)
	}
}

func TestSocialUsecase_FollowNotifications(t *testing.T) {
	ctx := context.Background()
	const pub, priv = int64(2), int64(3)
	uc, notifier := newSocialUC(pub, priv)

	// Following the public user accepts instantly and emits NO request notification.
	if _, err := uc.Follow(ctx, priv, pub); err != nil {
		t.Fatalf("follow public: %v", err)
	}
	if len(notifier.created) != 0 {
		t.Fatalf("public follow should not notify, got %+v", notifier.created)
	}

	// Following the private user is pending → a follow_request to the followee.
	if _, err := uc.Follow(ctx, pub, priv); err != nil {
		t.Fatalf("follow private: %v", err)
	}
	if len(notifier.created) != 1 {
		t.Fatalf("pending follow should emit one notification, got %d", len(notifier.created))
	}
	req := notifier.created[0]
	if req.Type != entity.NotifyFollowRequest || req.UserID != priv ||
		req.ActorID == nil || *req.ActorID != pub || req.Channel != entity.ChannelInApp ||
		req.DedupeKey == "" || req.Payload == "" {
		t.Fatalf("unexpected follow_request notification: %+v", req)
	}

	// Repeat follow is deduped — still a single notification.
	if _, err := uc.Follow(ctx, pub, priv); err != nil {
		t.Fatalf("repeat follow: %v", err)
	}
	if len(notifier.created) != 1 {
		t.Fatalf("repeat follow should not duplicate the notification, got %d", len(notifier.created))
	}

	// Approval emits a follow_accepted to the follower.
	if err := uc.Approve(ctx, priv, pub); err != nil {
		t.Fatalf("approve: %v", err)
	}
	var accepted *entity.Notification
	for i := range notifier.created {
		if notifier.created[i].Type == entity.NotifyFollowAccepted {
			accepted = &notifier.created[i]
		}
	}
	if accepted == nil || accepted.UserID != pub || accepted.ActorID == nil || *accepted.ActorID != priv ||
		accepted.Channel != entity.ChannelInApp || accepted.Payload == "" {
		t.Fatalf("unexpected follow_accepted notification: %+v", accepted)
	}

	// Unfollowing clears both follow notifications so a later re-follow re-fires.
	if err := uc.Unfollow(ctx, pub, priv); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	if len(notifier.created) != 0 {
		t.Fatalf("unfollow should clear follow notifications, got %+v", notifier.created)
	}
	if _, err := uc.Follow(ctx, pub, priv); err != nil {
		t.Fatalf("re-follow: %v", err)
	}
	if len(notifier.created) != 1 {
		t.Fatalf("re-follow after unfollow should re-fire, got %d", len(notifier.created))
	}
}

func TestSocialUsecase_CanViewProfile(t *testing.T) {
	ctx := context.Background()
	const pub, priv, stranger = int64(2), int64(3), int64(4)
	uc, _ := newSocialUC(pub, priv)

	// Owner always sees themselves.
	if ok, _ := uc.CanViewProfile(ctx, priv, priv); !ok {
		t.Fatal("owner should view own private profile")
	}
	// Anyone sees a public profile.
	if ok, _ := uc.CanViewProfile(ctx, stranger, pub); !ok {
		t.Fatal("stranger should view public profile")
	}
	// A stranger cannot see a private profile.
	if ok, _ := uc.CanViewProfile(ctx, stranger, priv); ok {
		t.Fatal("stranger should NOT view private profile")
	}
	// Pending follower still cannot see a private profile.
	if _, err := uc.Follow(ctx, stranger, priv); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if ok, _ := uc.CanViewProfile(ctx, stranger, priv); ok {
		t.Fatal("pending follower should NOT view private profile")
	}
	// After approval the accepted follower can see it.
	if err := uc.Approve(ctx, priv, stranger); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if ok, _ := uc.CanViewProfile(ctx, stranger, priv); !ok {
		t.Fatal("accepted follower should view private profile")
	}
}

func TestSocialUsecase_AutoAcceptOnPublic(t *testing.T) {
	ctx := context.Background()
	const pub, priv, s1, s2 = int64(2), int64(3), int64(5), int64(6)
	uc, _ := newSocialUC(pub, priv)

	// Two pending requests to the private user.
	for _, s := range []int64{s1, s2} {
		if _, err := uc.Follow(ctx, s, priv); err != nil {
			t.Fatalf("follow: %v", err)
		}
	}
	if len(mustIncoming(t, uc, priv)) != 2 {
		t.Fatal("expected two pending requests")
	}
	// Going public accepts them all.
	if err := uc.OnProfileMadePublic(ctx, priv); err != nil {
		t.Fatalf("on public: %v", err)
	}
	if n := len(mustIncoming(t, uc, priv)); n != 0 {
		t.Fatalf("pending after going public: %d", n)
	}
	if followers, _, _ := uc.Followers(ctx, priv, 10, 0); len(followers) != 2 {
		t.Fatalf("accepted followers after going public: %d", len(followers))
	}
}

func mustIncoming(t *testing.T, uc *usecase.SocialUsecase, id int64) []entity.User {
	t.Helper()
	in, err := uc.Incoming(context.Background(), id)
	if err != nil {
		t.Fatalf("incoming: %v", err)
	}
	return in
}
