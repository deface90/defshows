package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
)

// Social usecase errors.
var (
	ErrSelfFollow   = errors.New("usecase: cannot follow yourself")
	ErrUserNotFound = errors.New("usecase: user not found")
	ErrSelfBlock    = errors.New("usecase: cannot block yourself")
	ErrBlocked      = errors.New("usecase: blocked")
)

// SocialRepo is the follow-storage dependency of SocialUsecase.
type SocialRepo interface {
	CreateFollow(ctx context.Context, f *entity.Follow) error
	CreateFollowGuarded(ctx context.Context, f *entity.Follow) error
	GetFollow(ctx context.Context, followerID, followeeID int64) (*entity.Follow, error)
	FollowStatesFor(ctx context.Context, followerID int64, targetIDs []int64) (map[int64]entity.FollowStatus, error)
	DeleteFollow(ctx context.Context, followerID, followeeID int64) error
	SetFollowStatus(ctx context.Context, followerID, followeeID int64, status entity.FollowStatus) error
	AcceptAllPending(ctx context.Context, followeeID int64) error
	ListFollowers(ctx context.Context, userID int64, limit, offset int) ([]entity.User, int64, error)
	ListFollowing(ctx context.Context, userID int64, limit, offset int) ([]entity.User, int64, error)
	ListIncoming(ctx context.Context, userID int64) ([]entity.User, error)
	CountFollowers(ctx context.Context, userID int64) (int64, error)
	CountFollowing(ctx context.Context, userID int64) (int64, error)

	CreateBlock(ctx context.Context, b *entity.Block) error
	DeleteBlock(ctx context.Context, blockerID, blockedID int64) error
	ListBlocked(ctx context.Context, blockerID int64, limit, offset int) ([]entity.User, int64, error)
	IsBlockedEither(ctx context.Context, a, b int64) (bool, error)
	BlockedIDsEither(ctx context.Context, userID int64) ([]int64, error)
	DeleteFollowEither(ctx context.Context, a, b int64) error
}

// SocialUserRepo looks up users for visibility and existence checks.
type SocialUserRepo interface {
	FindUserByID(ctx context.Context, id int64) (*entity.User, error)
}

// FollowNotifier enqueues (and clears) follow notifications, and reads the recipient's
// prefs to honour their social-follows toggle. It is the notification repository in
// production. A follow notification is one feed event delivered to every channel the
// recipient has linked (telegram/apns/fcm); with none linked it is feed-only. Either way
// it surfaces in the in-app feed.
type FollowNotifier interface {
	EnqueueNotification(ctx context.Context, n *entity.Notification, channels []string) (bool, error)
	DeleteByDedupeKeys(ctx context.Context, keys ...string) error
	GetPrefs(ctx context.Context, userID int64) (*entity.NotificationPref, error)
}

// SocialUsecase implements follows, follow-gated visibility, and (later) feeds.
type SocialUsecase struct {
	repo     SocialRepo
	users    SocialUserRepo
	notifier FollowNotifier
}

// NewSocialUsecase creates a SocialUsecase.
func NewSocialUsecase(repo SocialRepo, users SocialUserRepo, notifier FollowNotifier) *SocialUsecase {
	return &SocialUsecase{repo: repo, users: users, notifier: notifier}
}

func followRequestDedupe(follower, followee int64) string {
	return fmt.Sprintf("follow_request:%d:%d", follower, followee)
}

func followAcceptedDedupe(follower, followee int64) string {
	return fmt.Sprintf("follow_accepted:%d:%d", follower, followee)
}

func followNewDedupe(follower, followee int64) string {
	return fmt.Sprintf("follow_new:%d:%d", follower, followee)
}

// followChannels lists every delivery channel the recipient has linked. An empty result
// means the notification is feed-only (it still surfaces in the in-app feed).
func followChannels(u *entity.User) []string {
	if u == nil {
		return nil
	}
	channels := []string{}
	if u.TelegramChatID != nil {
		channels = append(channels, "telegram")
	}
	if u.APNsDeviceToken != nil {
		channels = append(channels, "apns")
	}
	if u.FCMDeviceToken != nil {
		channels = append(channels, "fcm")
	}
	return channels
}

// actorName returns a label for a user to embed in a notification payload. It matches the
// name shown elsewhere (display name → email local part → "Пользователь #id").
func actorName(u *entity.User) string {
	return u.DisplayLabel()
}

// emitFollow enqueues a follow notification for recipientID triggered by actorID.
// Best-effort (the follow edge is already persisted): errors are swallowed. It honours
// the recipient's social-follows toggle and routes the row to their linked channel.
func (uc *SocialUsecase) emitFollow(ctx context.Context, recipientID, actorID int64, notifyType, dedupe, payload string) {
	if uc.notifier == nil {
		return
	}
	if prefs, err := uc.notifier.GetPrefs(ctx, recipientID); err == nil && prefs != nil && !prefs.SocialFollows {
		return
	}
	recipient, _ := uc.users.FindUserByID(ctx, recipientID)
	n := &entity.Notification{
		UserID:    recipientID,
		Type:      notifyType,
		ActorID:   &actorID,
		DedupeKey: dedupe,
		Payload:   payload,
	}
	_, _ = uc.notifier.EnqueueNotification(ctx, n, followChannels(recipient))
}

// emitFollowRequest notifies the followee that someone requested to follow them.
func (uc *SocialUsecase) emitFollowRequest(ctx context.Context, followerID, followeeID int64) {
	actor, _ := uc.users.FindUserByID(ctx, followerID)
	uc.emitFollow(ctx, followeeID, followerID, entity.NotifyFollowRequest,
		followRequestDedupe(followerID, followeeID),
		fmt.Sprintf("%s хочет на вас подписаться", actorName(actor)))
}

// emitFollowNew notifies the followee that someone (a public follow is accepted
// immediately, so there is no request to approve) started following them.
func (uc *SocialUsecase) emitFollowNew(ctx context.Context, followerID, followeeID int64) {
	actor, _ := uc.users.FindUserByID(ctx, followerID)
	uc.emitFollow(ctx, followeeID, followerID, entity.NotifyFollowNew,
		followNewDedupe(followerID, followeeID),
		fmt.Sprintf("%s подписался на вас", actorName(actor)))
}

// emitFollowAccepted notifies the follower that their request was accepted.
func (uc *SocialUsecase) emitFollowAccepted(ctx context.Context, followerID, followeeID int64) {
	actor, _ := uc.users.FindUserByID(ctx, followeeID)
	uc.emitFollow(ctx, followerID, followeeID, entity.NotifyFollowAccepted,
		followAcceptedDedupe(followerID, followeeID),
		fmt.Sprintf("%s принял вашу заявку на подписку", actorName(actor)))
}

// clearFollowNotifications removes both follow notifications for a pair so a later
// re-follow re-fires them (chosen over a nonce in the dedupe key).
func (uc *SocialUsecase) clearFollowNotifications(ctx context.Context, followerID, followeeID int64) {
	if uc.notifier == nil {
		return
	}
	_ = uc.notifier.DeleteByDedupeKeys(ctx,
		followRequestDedupe(followerID, followeeID),
		followNewDedupe(followerID, followeeID),
		followAcceptedDedupe(followerID, followeeID))
}

// Follow creates (or returns the existing) follow edge. Following a public user is
// accepted immediately; following a private user creates a pending request. Idempotent.
func (uc *SocialUsecase) Follow(ctx context.Context, followerID, followeeID int64) (*entity.Follow, error) {
	if followerID == followeeID {
		return nil, ErrSelfFollow
	}
	blocked, err := uc.repo.IsBlockedEither(ctx, followerID, followeeID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrBlocked
	}
	target, err := uc.users.FindUserByID(ctx, followeeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if existing, err := uc.repo.GetFollow(ctx, followerID, followeeID); err == nil {
		return existing, nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	status := entity.FollowPending
	if target.IsPublic {
		status = entity.FollowAccepted
	}
	f := &entity.Follow{FollowerID: followerID, FolloweeID: followeeID, Status: status}
	// CreateFollowGuarded re-checks the block inside the same transaction as the insert,
	// so a concurrent Block that committed its block row after the check above cannot leave
	// a surviving edge: the insert is refused with ErrBlocked. This closes the Follow/Block
	// TOCTOU window at the source (the home-feed SQL exclusion is a defence-in-depth backstop).
	if err := uc.repo.CreateFollowGuarded(ctx, f); err != nil {
		if errors.Is(err, repository.ErrBlocked) {
			return nil, ErrBlocked
		}
		// A concurrent insert may have won the race — return the winner.
		if errors.Is(err, repository.ErrConflict) {
			return uc.repo.GetFollow(ctx, followerID, followeeID)
		}
		return nil, err
	}
	if status == entity.FollowPending {
		uc.emitFollowRequest(ctx, followerID, followeeID)
	} else {
		uc.emitFollowNew(ctx, followerID, followeeID)
	}
	return f, nil
}

// Unfollow removes a follow edge (also cancels a pending request). Idempotent.
func (uc *SocialUsecase) Unfollow(ctx context.Context, followerID, followeeID int64) error {
	err := uc.repo.DeleteFollow(ctx, followerID, followeeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	uc.clearFollowNotifications(ctx, followerID, followeeID)
	return nil
}

// Approve accepts a pending request from followerID to ownerID and notifies the
// follower. If a block exists in either direction the request's edge has already been
// torn down, so Approve is a no-op (rather than surfacing a confusing 404) — but a
// genuinely absent edge still returns the repo's ErrNotFound so the handler's existing
// "no pending request" 404 is preserved.
func (uc *SocialUsecase) Approve(ctx context.Context, ownerID, followerID int64) error {
	if blocked, err := uc.repo.IsBlockedEither(ctx, ownerID, followerID); err != nil {
		return err
	} else if blocked {
		return nil
	}
	if err := uc.repo.SetFollowStatus(ctx, followerID, ownerID, entity.FollowAccepted); err != nil {
		return err
	}
	uc.emitFollowAccepted(ctx, followerID, ownerID)
	return nil
}

// Reject deletes a pending request from followerID to ownerID and clears its
// notification.
func (uc *SocialUsecase) Reject(ctx context.Context, ownerID, followerID int64) error {
	if err := uc.repo.DeleteFollow(ctx, followerID, ownerID); err != nil {
		return err
	}
	uc.clearFollowNotifications(ctx, followerID, ownerID)
	return nil
}

// RemoveFollower ejects followerID from ownerID's followers (or a pending requester):
// it deletes the follower→owner edge and clears its follow notifications. Idempotent:
// a no-op when there is no such edge.
func (uc *SocialUsecase) RemoveFollower(ctx context.Context, ownerID, followerID int64) error {
	err := uc.repo.DeleteFollow(ctx, followerID, ownerID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	uc.clearFollowNotifications(ctx, followerID, ownerID)
	return nil
}

// Incoming returns the users with a pending request to the owner.
func (uc *SocialUsecase) Incoming(ctx context.Context, ownerID int64) ([]entity.User, error) {
	return uc.repo.ListIncoming(ctx, ownerID)
}

// Followers returns the user's accepted followers (paginated).
func (uc *SocialUsecase) Followers(ctx context.Context, userID int64, limit, offset int) ([]entity.User, int64, error) {
	return uc.repo.ListFollowers(ctx, userID, limit, offset)
}

// Following returns the users the user follows (paginated).
func (uc *SocialUsecase) Following(ctx context.Context, userID int64, limit, offset int) ([]entity.User, int64, error) {
	return uc.repo.ListFollowing(ctx, userID, limit, offset)
}

// Counts returns the user's follower/following totals.
func (uc *SocialUsecase) Counts(ctx context.Context, userID int64) (followers, following int64, err error) {
	followers, err = uc.repo.CountFollowers(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	following, err = uc.repo.CountFollowing(ctx, userID)
	return followers, following, err
}

// FollowState reports the viewer's relationship to a target: "none", "pending" or
// "accepted". A viewer is never "following" themselves.
func (uc *SocialUsecase) FollowState(ctx context.Context, viewerID, targetID int64) (string, error) {
	if viewerID == 0 || viewerID == targetID {
		return "none", nil
	}
	f, err := uc.repo.GetFollow(ctx, viewerID, targetID)
	if errors.Is(err, repository.ErrNotFound) {
		return "none", nil
	}
	if err != nil {
		return "", err
	}
	return string(f.Status), nil
}

// FollowStates reports the viewer's relationship ("none"/"pending"/"accepted") to each
// target id, batched to a single lookup. Targets with no edge (and the viewer's own id,
// or a guest viewer) map to "none".
func (uc *SocialUsecase) FollowStates(ctx context.Context, viewerID int64, targetIDs []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(targetIDs))
	for _, id := range targetIDs {
		out[id] = "none"
	}
	if viewerID == 0 || len(targetIDs) == 0 {
		return out, nil
	}
	edges, err := uc.repo.FollowStatesFor(ctx, viewerID, targetIDs)
	if err != nil {
		return nil, err
	}
	for id, status := range edges {
		out[id] = string(status)
	}
	return out, nil
}

// CanViewProfile reports whether viewer may see target's collection and activity:
// the owner, a public profile, or an accepted follower.
func (uc *SocialUsecase) CanViewProfile(ctx context.Context, viewerID, targetID int64) (bool, error) {
	if viewerID != 0 && viewerID == targetID {
		return true, nil
	}
	if viewerID != 0 {
		blocked, err := uc.repo.IsBlockedEither(ctx, viewerID, targetID)
		if err != nil {
			return false, err
		}
		if blocked {
			return false, nil
		}
	}
	target, err := uc.users.FindUserByID(ctx, targetID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, ErrUserNotFound
		}
		return false, err
	}
	if target.IsPublic {
		return true, nil
	}
	if viewerID == 0 {
		return false, nil
	}
	f, err := uc.repo.GetFollow(ctx, viewerID, targetID)
	if errors.Is(err, repository.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return f.Status == entity.FollowAccepted, nil
}

// OnProfileMadePublic accepts all pending requests once a user opts their profile
// public. Call this after persisting the visibility change.
func (uc *SocialUsecase) OnProfileMadePublic(ctx context.Context, userID int64) error {
	return uc.repo.AcceptAllPending(ctx, userID)
}

// Block establishes a block from blockerID to blockedID: a mutual visibility cut-off.
// It tears down follow edges in both directions and clears their follow notifications so
// no stale request/accepted lingers. Idempotent: re-blocking an already-blocked user is a
// no-op. Blocking is intentionally not reported to the blocked user.
func (uc *SocialUsecase) Block(ctx context.Context, blockerID, blockedID int64) error {
	if blockerID == blockedID {
		return ErrSelfBlock
	}
	if _, err := uc.users.FindUserByID(ctx, blockedID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	b := &entity.Block{BlockerID: blockerID, BlockedID: blockedID}
	if err := uc.repo.CreateBlock(ctx, b); err != nil && !errors.Is(err, repository.ErrConflict) {
		return err
	}
	if err := uc.repo.DeleteFollowEither(ctx, blockerID, blockedID); err != nil {
		return err
	}
	uc.clearFollowNotifications(ctx, blockerID, blockedID)
	uc.clearFollowNotifications(ctx, blockedID, blockerID)
	return nil
}

// Unblock removes the directed block from blockerID to blockedID. Idempotent: no error
// when there was no block.
func (uc *SocialUsecase) Unblock(ctx context.Context, blockerID, blockedID int64) error {
	err := uc.repo.DeleteBlock(ctx, blockerID, blockedID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	return err
}

// Blocked returns the users the given user has blocked (paginated).
func (uc *SocialUsecase) Blocked(ctx context.Context, userID int64, limit, offset int) ([]entity.User, int64, error) {
	return uc.repo.ListBlocked(ctx, userID, limit, offset)
}

// BlockedIDs returns the ids of every user involved in a block with the given user in
// either direction — the exclusion set for directory search.
func (uc *SocialUsecase) BlockedIDs(ctx context.Context, userID int64) ([]int64, error) {
	return uc.repo.BlockedIDsEither(ctx, userID)
}

// IsBlockedEither reports whether a block exists between the two users in either
// direction.
func (uc *SocialUsecase) IsBlockedEither(ctx context.Context, a, b int64) (bool, error) {
	return uc.repo.IsBlockedEither(ctx, a, b)
}
