package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/deface90/defshows/backend/internal/service/entity"
)

// SocialRepository handles follow relationships.
type SocialRepository struct {
	db *gorm.DB
}

// NewSocialRepository creates a SocialRepository.
func NewSocialRepository(db *gorm.DB) *SocialRepository {
	return &SocialRepository{db: db}
}

// CreateFollow inserts a follow edge. Returns ErrConflict if one already exists.
func (r *SocialRepository) CreateFollow(ctx context.Context, f *entity.Follow) error {
	if err := r.db.WithContext(ctx).Create(f).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrConflict
		}
		return err
	}
	return nil
}

// CreateFollowGuarded inserts a follow edge inside a transaction that first re-checks
// for a block in either direction, closing the Follow/Block TOCTOU window: a concurrent
// Block that commits its block row before this transaction's block re-check will cause
// the insert to be refused (ErrBlocked). Returns ErrConflict if the edge already exists.
// This is the root fix for the race where a follow that passed an earlier (non-transactional)
// block check could re-insert an edge after Block tore it down.
func (r *SocialRepository) CreateFollowGuarded(ctx context.Context, f *entity.Follow) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&entity.Block{}).
			Where("(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)",
				f.FollowerID, f.FolloweeID, f.FolloweeID, f.FollowerID).
			Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return ErrBlocked
		}
		if err := tx.Create(f).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrConflict
			}
			return err
		}
		return nil
	})
}

// GetFollow returns the follow edge from follower to followee, or ErrNotFound.
func (r *SocialRepository) GetFollow(ctx context.Context, followerID, followeeID int64) (*entity.Follow, error) {
	var f entity.Follow
	err := r.db.WithContext(ctx).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// FollowStatesFor returns followerID's edge status to each target that has an edge
// (follower_id = followerID AND followee_id IN targetIDs). Targets with no edge are
// absent from the map. A zero follower or empty target list yields an empty map with
// no query (gorm would otherwise emit `IN (NULL)`).
func (r *SocialRepository) FollowStatesFor(ctx context.Context, followerID int64, targetIDs []int64) (map[int64]entity.FollowStatus, error) {
	out := make(map[int64]entity.FollowStatus, len(targetIDs))
	if followerID == 0 || len(targetIDs) == 0 {
		return out, nil
	}
	var edges []entity.Follow
	if err := r.db.WithContext(ctx).
		Where("follower_id = ? AND followee_id IN ?", followerID, targetIDs).
		Find(&edges).Error; err != nil {
		return nil, err
	}
	for i := range edges {
		out[edges[i].FolloweeID] = edges[i].Status
	}
	return out, nil
}

// DeleteFollow removes the follow edge. Returns ErrNotFound if there was none.
func (r *SocialRepository) DeleteFollow(ctx context.Context, followerID, followeeID int64) error {
	res := r.db.WithContext(ctx).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Delete(&entity.Follow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// SetFollowStatus updates an existing edge's status. Returns ErrNotFound if none.
func (r *SocialRepository) SetFollowStatus(ctx context.Context, followerID, followeeID int64, status entity.FollowStatus) error {
	res := r.db.WithContext(ctx).Model(&entity.Follow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Updates(map[string]any{"status": status, "updated_at": gorm.Expr("now()")})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// AcceptAllPending flips every pending request to the given followee to accepted.
// Used when a user switches their profile to public.
func (r *SocialRepository) AcceptAllPending(ctx context.Context, followeeID int64) error {
	return r.db.WithContext(ctx).Model(&entity.Follow{}).
		Where("followee_id = ? AND status = ?", followeeID, entity.FollowPending).
		Updates(map[string]any{"status": entity.FollowAccepted, "updated_at": gorm.Expr("now()")}).Error
}

// ListFollowers returns accepted followers of the user (paginated), newest first.
func (r *SocialRepository) ListFollowers(ctx context.Context, userID int64, limit, offset int) ([]entity.User, int64, error) {
	return r.listEdgeUsers(ctx,
		"JOIN follows ON follows.follower_id = users.id",
		"follows.followee_id = ? AND follows.status = ?", userID, entity.FollowAccepted, limit, offset)
}

// ListFollowing returns the users this user follows with accepted status (paginated).
func (r *SocialRepository) ListFollowing(ctx context.Context, userID int64, limit, offset int) ([]entity.User, int64, error) {
	return r.listEdgeUsers(ctx,
		"JOIN follows ON follows.followee_id = users.id",
		"follows.follower_id = ? AND follows.status = ?", userID, entity.FollowAccepted, limit, offset)
}

// ListIncoming returns the users who have a pending follow request to the user.
func (r *SocialRepository) ListIncoming(ctx context.Context, userID int64) ([]entity.User, error) {
	users, _, err := r.listEdgeUsers(ctx,
		"JOIN follows ON follows.follower_id = users.id",
		"follows.followee_id = ? AND follows.status = ?", userID, entity.FollowPending, 0, 0)
	return users, err
}

// listEdgeUsers runs the shared join/count/page query for follower/following/incoming
// lists. limit <= 0 means "no limit" (used for the typically-small incoming list).
func (r *SocialRepository) listEdgeUsers(ctx context.Context, join, where string, id int64, status entity.FollowStatus, limit, offset int) ([]entity.User, int64, error) {
	base := r.db.WithContext(ctx).Model(&entity.User{}).Joins(join).Where(where, id, status)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := base.Order("follows.created_at DESC, users.id DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	var users []entity.User
	if err := q.Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// AcceptedFolloweeIDs returns the ids of users the given user follows with accepted
// status — the subscription set backing the home feed. Users involved in a block with
// the viewer in either direction are excluded in SQL, so the home feed never surfaces a
// blocked actor even if a follow edge lingers due to a Follow/Block race (a follow that
// passed its block check can re-insert an edge after Block tore it down).
func (r *SocialRepository) AcceptedFolloweeIDs(ctx context.Context, userID int64) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&entity.Follow{}).
		Where("follower_id = ? AND status = ?", userID, entity.FollowAccepted).
		Where(`followee_id NOT IN (
			SELECT blocked_id FROM blocks WHERE blocker_id = ?
			UNION
			SELECT blocker_id FROM blocks WHERE blocked_id = ?
		)`, userID, userID).
		Pluck("followee_id", &ids).Error
	return ids, err
}

// CountFollowers returns the number of accepted followers of the user.
func (r *SocialRepository) CountFollowers(ctx context.Context, userID int64) (int64, error) {
	return r.countEdges(ctx, "followee_id = ? AND status = ?", userID)
}

// CountFollowing returns the number of users the user follows (accepted).
func (r *SocialRepository) CountFollowing(ctx context.Context, userID int64) (int64, error) {
	return r.countEdges(ctx, "follower_id = ? AND status = ?", userID)
}

func (r *SocialRepository) countEdges(ctx context.Context, where string, id int64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.Follow{}).
		Where(where, id, entity.FollowAccepted).Count(&n).Error
	return n, err
}

// CreateBlock inserts a block edge. Returns ErrConflict if one already exists.
func (r *SocialRepository) CreateBlock(ctx context.Context, b *entity.Block) error {
	if err := r.db.WithContext(ctx).Create(b).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrConflict
		}
		return err
	}
	return nil
}

// DeleteBlock removes the directed block edge. Returns ErrNotFound if there was none.
func (r *SocialRepository) DeleteBlock(ctx context.Context, blockerID, blockedID int64) error {
	res := r.db.WithContext(ctx).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Delete(&entity.Block{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListBlocked returns the users the given user has blocked (paginated), newest first.
func (r *SocialRepository) ListBlocked(ctx context.Context, blockerID int64, limit, offset int) ([]entity.User, int64, error) {
	base := r.db.WithContext(ctx).Model(&entity.User{}).
		Joins("JOIN blocks ON blocks.blocked_id = users.id").
		Where("blocks.blocker_id = ?", blockerID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := base.Order("blocks.created_at DESC, users.id DESC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	var users []entity.User
	if err := q.Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// IsBlockedEither reports whether a block exists between the two users in either
// direction (a blocked b or b blocked a).
func (r *SocialRepository) IsBlockedEither(ctx context.Context, a, b int64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.Block{}).
		Where("(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)", a, b, b, a).
		Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// BlockedIDsEither returns the ids of every user involved in a block with the given
// user in either direction (users they blocked plus users who blocked them).
func (r *SocialRepository) BlockedIDsEither(ctx context.Context, userID int64) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&entity.Block{}).
		Where("blocker_id = ?", userID).
		Pluck("blocked_id", &ids).Error
	if err != nil {
		return nil, err
	}
	var incoming []int64
	err = r.db.WithContext(ctx).Model(&entity.Block{}).
		Where("blocked_id = ?", userID).
		Pluck("blocker_id", &incoming).Error
	if err != nil {
		return nil, err
	}
	return append(ids, incoming...), nil
}

// DeleteFollowEither removes both directed follow rows between the two users in one
// query. Not an error if none existed (used when a block tears down edges).
func (r *SocialRepository) DeleteFollowEither(ctx context.Context, a, b int64) error {
	return r.db.WithContext(ctx).
		Where("(follower_id = ? AND followee_id = ?) OR (follower_id = ? AND followee_id = ?)", a, b, b, a).
		Delete(&entity.Follow{}).Error
}
