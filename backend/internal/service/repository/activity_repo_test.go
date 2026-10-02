package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/testutil"
)

func TestActivityRepository(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows")
	ctx := context.Background()

	user := &entity.User{Email: strptr("activity@example.com"), Role: entity.RoleUser, Timezone: "UTC"}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	show := &entity.Show{TMDBID: 777, Title: "Show", AiringStatus: entity.AiringEnded}
	if err := gdb.Create(show).Error; err != nil {
		t.Fatalf("seed show: %v", err)
	}

	activity := repository.NewActivityRepository(gdb)

	// Standalone insert fills created_at and persists the row.
	if err := activity.InsertEvents(ctx, entity.ActivityEvent{
		UserID: user.ID, Type: entity.EventAddedShow, ShowID: show.ID,
	}); err != nil {
		t.Fatalf("insert events: %v", err)
	}
	var stored []entity.ActivityEvent
	if err := gdb.Where("user_id = ?", user.ID).Find(&stored).Error; err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(stored) != 1 || stored[0].Type != entity.EventAddedShow || stored[0].CreatedAt.IsZero() {
		t.Fatalf("unexpected stored event: %+v", stored)
	}

	// InsertEvents with no events is a no-op.
	if err := activity.InsertEvents(ctx); err != nil {
		t.Fatalf("empty insert: %v", err)
	}

	// Same-transaction atomicity: an event referencing a non-existent show (FK
	// violation) must roll back the tracking mutation it rode along with.
	tracking := repository.NewTrackingRepository(gdb)
	us := &entity.UserShow{UserID: user.ID, ShowID: show.ID, Status: entity.StatusWatching}
	badEvent := entity.ActivityEvent{UserID: user.ID, Type: entity.EventAddedShow, ShowID: 999999}
	if err := tracking.AddUserShow(ctx, us, badEvent); err == nil {
		t.Fatal("expected FK error from the bad event")
	}
	// The user_show insert rode the same failed transaction, so it must be gone.
	if _, err := tracking.GetUserShow(ctx, user.ID, show.ID); err != repository.ErrNotFound {
		t.Fatalf("user_show should have rolled back, got %v", err)
	}
	// And no stray event was written.
	var count int64
	gdb.Model(&entity.ActivityEvent{}).Where("show_id = ?", 999999).Count(&count)
	if count != 0 {
		t.Fatalf("bad event should have rolled back, found %d", count)
	}
}

func TestActivityRepository_KeysetFeeds(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows", "activity_events")
	ctx := context.Background()

	u1 := &entity.User{Email: strptr("u1@example.com"), Role: entity.RoleUser, Timezone: "UTC"}
	u2 := &entity.User{Email: strptr("u2@example.com"), Role: entity.RoleUser, Timezone: "UTC"}
	for _, u := range []*entity.User{u1, u2} {
		if err := gdb.Create(u).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	show := &entity.Show{TMDBID: 111, Title: "Show", AiringStatus: entity.AiringEnded}
	if err := gdb.Create(show).Error; err != nil {
		t.Fatalf("seed show: %v", err)
	}

	repo := repository.NewActivityRepository(gdb)
	base := time.Now().Truncate(time.Second)
	// u1: three events, newest last inserted; u2: one event.
	mk := func(user int64, offset time.Duration) entity.ActivityEvent {
		return entity.ActivityEvent{UserID: user, Type: entity.EventAddedShow, ShowID: show.ID, CreatedAt: base.Add(offset)}
	}
	if err := repo.InsertEvents(ctx,
		mk(u1.ID, -3*time.Hour), mk(u1.ID, -2*time.Hour), mk(u1.ID, -1*time.Hour),
		mk(u2.ID, -90*time.Minute),
	); err != nil {
		t.Fatalf("seed events: %v", err)
	}

	// Profile feed of u1: newest first, keyset paging covers all three without overlap.
	first, err := repo.ProfileEvents(ctx, u1.ID, nil, 2)
	if err != nil || len(first) != 2 {
		t.Fatalf("profile page1: len=%d err=%v", len(first), err)
	}
	if !first[0].CreatedAt.After(first[1].CreatedAt) {
		t.Fatalf("profile events not newest-first: %v, %v", first[0].CreatedAt, first[1].CreatedAt)
	}
	cur := &repository.FeedCursor{CreatedAt: first[1].CreatedAt, ID: first[1].ID}
	second, err := repo.ProfileEvents(ctx, u1.ID, cur, 2)
	if err != nil || len(second) != 1 {
		t.Fatalf("profile page2: len=%d err=%v", len(second), err)
	}
	if second[0].ID == first[0].ID || second[0].ID == first[1].ID {
		t.Fatalf("keyset page2 overlapped page1")
	}

	// Home feed over both users returns all four, newest first.
	home, err := repo.HomeEvents(ctx, []int64{u1.ID, u2.ID}, nil, 10)
	if err != nil || len(home) != 4 {
		t.Fatalf("home feed: len=%d err=%v", len(home), err)
	}
	for i := 1; i < len(home); i++ {
		if home[i-1].CreatedAt.Before(home[i].CreatedAt) {
			t.Fatalf("home feed not sorted desc at %d", i)
		}
	}
	// Empty followee set yields nothing.
	if rows, err := repo.HomeEvents(ctx, nil, nil, 10); err != nil || len(rows) != 0 {
		t.Fatalf("empty home feed: len=%d err=%v", len(rows), err)
	}
}
