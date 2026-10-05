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
	edges  map[[2]int64]*entity.Follow
	blocks map[[2]int64]bool // (blocker, blocked)
	// beforeGuardedInsert simulates a concurrent operation interleaving inside
	// CreateFollowGuarded's transaction (e.g. a Block committing its block row).
	beforeGuardedInsert func()
}

func newFakeSocialRepo() *fakeSocialRepo {
	return &fakeSocialRepo{edges: map[[2]int64]*entity.Follow{}, blocks: map[[2]int64]bool{}}
}

func (r *fakeSocialRepo) CreateBlock(_ context.Context, b *entity.Block) error {
	k := [2]int64{b.BlockerID, b.BlockedID}
	if r.blocks[k] {
		return repository.ErrConflict
	}
	r.blocks[k] = true
	return nil
}
func (r *fakeSocialRepo) DeleteBlock(_ context.Context, blocker, blocked int64) error {
	k := [2]int64{blocker, blocked}
	if !r.blocks[k] {
		return repository.ErrNotFound
	}
	delete(r.blocks, k)
	return nil
}
func (r *fakeSocialRepo) ListBlocked(_ context.Context, blocker int64, _, _ int) ([]entity.User, int64, error) {
	var out []entity.User
	for k := range r.blocks {
		if k[0] == blocker {
			out = append(out, entity.User{ID: k[1]})
		}
	}
	return out, int64(len(out)), nil
}
func (r *fakeSocialRepo) IsBlockedEither(_ context.Context, a, b int64) (bool, error) {
	return r.blocks[[2]int64{a, b}] || r.blocks[[2]int64{b, a}], nil
}
func (r *fakeSocialRepo) BlockedIDsEither(_ context.Context, userID int64) ([]int64, error) {
	var out []int64
	for k := range r.blocks {
		switch userID {
		case k[0]:
			out = append(out, k[1])
		case k[1]:
			out = append(out, k[0])
		}
	}
	return out, nil
}
func (r *fakeSocialRepo) DeleteFollowEither(_ context.Context, a, b int64) error {
	delete(r.edges, [2]int64{a, b})
	delete(r.edges, [2]int64{b, a})
	return nil
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

// beforeGuardedInsert, if set, runs inside CreateFollowGuarded after the block re-check
// but could just as well model a block committed before it — tests set it to simulate a
// concurrent Block interleaving with Follow. It mirrors the production transaction, which
// re-checks the block atomically with the insert.
func (r *fakeSocialRepo) CreateFollowGuarded(ctx context.Context, f *entity.Follow) error {
	if r.beforeGuardedInsert != nil {
		r.beforeGuardedInsert()
	}
	// Re-check the block atomically with the insert (production does this in one tx).
	if r.blocks[[2]int64{f.FollowerID, f.FolloweeID}] || r.blocks[[2]int64{f.FolloweeID, f.FollowerID}] {
		return repository.ErrBlocked
	}
	return r.CreateFollow(ctx, f)
}
func (r *fakeSocialRepo) GetFollow(_ context.Context, follower, followee int64) (*entity.Follow, error) {
	if f, ok := r.edges[[2]int64{follower, followee}]; ok {
		cp := *f
		return &cp, nil
	}
	return nil, repository.ErrNotFound
}
func (r *fakeSocialRepo) FollowStatesFor(_ context.Context, follower int64, targetIDs []int64) (map[int64]entity.FollowStatus, error) {
	out := map[int64]entity.FollowStatus{}
	for _, t := range targetIDs {
		if f, ok := r.edges[[2]int64{follower, t}]; ok {
			out[t] = f.Status
		}
	}
	return out, nil
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
	uc, _, notifier := newSocialUCWithRepo(public, private)
	return uc, notifier
}

func newSocialUCWithRepo(public, private int64) (*usecase.SocialUsecase, *fakeSocialRepo, *fakeFollowNotifier) {
	users := fakeSocialUsers{users: map[int64]*entity.User{
		public:  {ID: public, IsPublic: true},
		private: {ID: private, IsPublic: false},
	}}
	notifier := &fakeFollowNotifier{}
	repo := newFakeSocialRepo()
	return usecase.NewSocialUsecase(repo, users, notifier), repo, notifier
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

func TestSocialUsecase_FollowStates(t *testing.T) {
	ctx := context.Background()
	const viewer, pub, priv = int64(1), int64(2), int64(3)
	uc, _ := newSocialUC(pub, priv)

	// viewer → pub resolves to accepted (public target); viewer → priv stays pending.
	if _, err := uc.Follow(ctx, viewer, pub); err != nil {
		t.Fatalf("follow pub: %v", err)
	}
	if _, err := uc.Follow(ctx, viewer, priv); err != nil {
		t.Fatalf("follow priv: %v", err)
	}

	states, err := uc.FollowStates(ctx, viewer, []int64{pub, priv, 999, viewer})
	if err != nil {
		t.Fatalf("FollowStates: %v", err)
	}
	// Every requested id is present; edges reflect status, no-edge/self default to none.
	if states[pub] != "accepted" || states[priv] != "pending" {
		t.Fatalf("edges: pub=%s priv=%s", states[pub], states[priv])
	}
	if states[999] != "none" || states[viewer] != "none" {
		t.Fatalf("defaults: 999=%s self=%s", states[999], states[viewer])
	}
	if len(states) != 4 {
		t.Fatalf("want 4 entries, got %d", len(states))
	}

	// A guest viewer (0) sees everyone as none without touching the repo.
	guest, err := uc.FollowStates(ctx, 0, []int64{pub, priv})
	if err != nil {
		t.Fatalf("guest FollowStates: %v", err)
	}
	if guest[pub] != "none" || guest[priv] != "none" {
		t.Fatalf("guest should see none: pub=%s priv=%s", guest[pub], guest[priv])
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

func TestSocialUsecase_Block(t *testing.T) {
	ctx := context.Background()
	const pub, priv = int64(2), int64(3)

	// Self-block rejected.
	uc, _ := newSocialUC(pub, priv)
	if err := uc.Block(ctx, pub, pub); !errors.Is(err, usecase.ErrSelfBlock) {
		t.Fatalf("self-block: want ErrSelfBlock, got %v", err)
	}
	// Unknown target rejected.
	if err := uc.Block(ctx, pub, 999); !errors.Is(err, usecase.ErrUserNotFound) {
		t.Fatalf("unknown target: want ErrUserNotFound, got %v", err)
	}

	// Block tears down both follow edges and clears both pairs' notifications.
	uc, notifier := newSocialUC(pub, priv)
	if _, err := uc.Follow(ctx, pub, priv); err != nil { // pub -> priv (pending, notifies priv)
		t.Fatalf("follow pub->priv: %v", err)
	}
	if _, err := uc.Follow(ctx, priv, pub); err != nil { // priv -> pub (accepted)
		t.Fatalf("follow priv->pub: %v", err)
	}
	if len(notifier.created) == 0 {
		t.Fatal("expected a follow_request notification before block")
	}
	if err := uc.Block(ctx, pub, priv); err != nil {
		t.Fatalf("block: %v", err)
	}
	if state, _ := uc.FollowState(ctx, pub, priv); state != "none" {
		t.Fatalf("pub->priv edge should be gone after block, got %s", state)
	}
	if state, _ := uc.FollowState(ctx, priv, pub); state != "none" {
		t.Fatalf("priv->pub edge should be gone after block, got %s", state)
	}
	if len(notifier.created) != 0 {
		t.Fatalf("block should clear follow notifications, got %+v", notifier.created)
	}

	// Re-block is idempotent.
	if err := uc.Block(ctx, pub, priv); err != nil {
		t.Fatalf("re-block should be idempotent: %v", err)
	}

	// Re-follow after block is rejected in both directions.
	if _, err := uc.Follow(ctx, pub, priv); !errors.Is(err, usecase.ErrBlocked) {
		t.Fatalf("follow by blocker: want ErrBlocked, got %v", err)
	}
	if _, err := uc.Follow(ctx, priv, pub); !errors.Is(err, usecase.ErrBlocked) {
		t.Fatalf("follow by blocked: want ErrBlocked, got %v", err)
	}

	// Blocked list contains the blocked user.
	blocked, total, err := uc.Blocked(ctx, pub, 10, 0)
	if err != nil || total != 1 || len(blocked) != 1 || blocked[0].ID != priv {
		t.Fatalf("blocked list: %v total=%d %+v", err, total, blocked)
	}

	// Unblock restores viewability for a public target and re-follow works again.
	if err := uc.Unblock(ctx, pub, priv); err != nil {
		t.Fatalf("unblock: %v", err)
	}
	if err := uc.Unblock(ctx, pub, priv); err != nil {
		t.Fatalf("second unblock should be no-op: %v", err)
	}
	if ok, _ := uc.CanViewProfile(ctx, priv, pub); !ok {
		t.Fatal("after unblock, blocked user should view the public profile again")
	}
	if _, err := uc.Follow(ctx, priv, pub); err != nil {
		t.Fatalf("re-follow after unblock: %v", err)
	}
}

// TestSocialUsecase_Follow_BlockRace simulates a Block committing between Follow's early
// block check and its edge insert. Because the insert goes through CreateFollowGuarded,
// which re-checks the block atomically with the insert, the follow is refused (ErrBlocked)
// and no surviving edge/notification is left for a later Unblock to resurrect.
func TestSocialUsecase_Follow_BlockRace(t *testing.T) {
	ctx := context.Background()
	const pub, other = int64(2), int64(3)
	uc, repo, notifier := newSocialUCWithRepo(pub, other)

	// Model the race: a Block for (other blocks pub) commits inside the window of the
	// guarded insert (after Follow's early check passed). The guard must see it.
	repo.beforeGuardedInsert = func() {
		repo.blocks[[2]int64{other, pub}] = true
	}

	if _, err := uc.Follow(ctx, pub, other); !errors.Is(err, usecase.ErrBlocked) {
		t.Fatalf("follow racing a block: want ErrBlocked, got %v", err)
	}
	// No surviving edge in either direction.
	if state, _ := uc.FollowState(ctx, pub, other); state != "none" {
		t.Fatalf("no edge should survive the race, got state %q", state)
	}
	// No follow_request notification was emitted (insert never completed).
	if len(notifier.created) != 0 {
		t.Fatalf("no notification should be emitted when the follow is refused, got %+v", notifier.created)
	}
	// A later Unblock cannot resurrect a relationship that never persisted.
	if err := uc.Unblock(ctx, other, pub); err != nil {
		t.Fatalf("unblock: %v", err)
	}
	if state, _ := uc.FollowState(ctx, pub, other); state != "none" {
		t.Fatalf("after unblock, still no edge should exist, got state %q", state)
	}
}

func TestSocialUsecase_CanViewProfile_Blocked(t *testing.T) {
	ctx := context.Background()
	const pub, other = int64(2), int64(3)

	// A blocks the public user B → neither can view the other (either direction).
	uc, _ := newSocialUC(pub, other)
	if err := uc.Block(ctx, other, pub); err != nil {
		t.Fatalf("block: %v", err)
	}
	if ok, _ := uc.CanViewProfile(ctx, other, pub); ok {
		t.Fatal("blocker should NOT view the blocked user's profile")
	}
	if ok, _ := uc.CanViewProfile(ctx, pub, other); ok {
		t.Fatal("blocked user should NOT view the blocker's profile")
	}
	// A guest (viewer 0) is unaffected by the block and sees the public profile.
	if ok, _ := uc.CanViewProfile(ctx, 0, pub); !ok {
		t.Fatal("guest should still view the public profile")
	}
}

func TestSocialUsecase_Approve_BlockedIsNoop(t *testing.T) {
	ctx := context.Background()
	const pub, priv = int64(2), int64(3)
	uc, _ := newSocialUC(pub, priv)

	// pub requests to follow priv (pending), then priv blocks pub which tears the edge.
	if _, err := uc.Follow(ctx, pub, priv); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if err := uc.Block(ctx, priv, pub); err != nil {
		t.Fatalf("block: %v", err)
	}
	// Approving the (now gone) request is a no-op, not a 404-producing ErrNotFound.
	if err := uc.Approve(ctx, priv, pub); err != nil {
		t.Fatalf("approve after block should be a no-op, got %v", err)
	}

	// A genuinely absent request (no block) still surfaces ErrNotFound.
	uc2, _ := newSocialUC(pub, priv)
	if err := uc2.Approve(ctx, priv, pub); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("approve with no pending request: want ErrNotFound, got %v", err)
	}
}

func TestSocialUsecase_RemoveFollower(t *testing.T) {
	ctx := context.Background()
	const owner, follower = int64(2), int64(3)

	// Removing an accepted follower drops them and clears the pair's follow notifications.
	uc, notifier := newSocialUC(owner, follower)
	if _, err := uc.Follow(ctx, follower, owner); err != nil { // follower -> owner (owner is public → accepted)
		t.Fatalf("follow: %v", err)
	}
	if followers, _, _ := uc.Followers(ctx, owner, 10, 0); len(followers) != 1 {
		t.Fatalf("expected one follower before remove, got %d", len(followers))
	}
	if err := uc.RemoveFollower(ctx, owner, follower); err != nil {
		t.Fatalf("remove follower: %v", err)
	}
	if followers, _, _ := uc.Followers(ctx, owner, 10, 0); len(followers) != 0 {
		t.Fatalf("follower should be gone after remove, got %d", len(followers))
	}
	if len(notifier.created) != 0 {
		t.Fatalf("remove should clear follow notifications, got %+v", notifier.created)
	}

	// Removing again (none present) is a no-op.
	if err := uc.RemoveFollower(ctx, owner, follower); err != nil {
		t.Fatalf("second remove should be a no-op: %v", err)
	}

	// A pending requester is also removable (equivalent to reject). Here the private
	// user is the owner, so a follow request to them stays pending.
	const pubUser, privOwner = int64(2), int64(3)
	uc2, notifier2 := newSocialUC(pubUser, privOwner)
	if _, err := uc2.Follow(ctx, pubUser, privOwner); err != nil { // pubUser -> privOwner stays pending
		t.Fatalf("pending follow: %v", err)
	}
	if len(mustIncoming(t, uc2, privOwner)) != 1 {
		t.Fatalf("expected one pending requester before remove")
	}
	if err := uc2.RemoveFollower(ctx, privOwner, pubUser); err != nil {
		t.Fatalf("remove pending requester: %v", err)
	}
	if len(mustIncoming(t, uc2, privOwner)) != 0 {
		t.Fatalf("pending requester should be gone after remove")
	}
	if len(notifier2.created) != 0 {
		t.Fatalf("remove should clear the pending request notification, got %+v", notifier2.created)
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
