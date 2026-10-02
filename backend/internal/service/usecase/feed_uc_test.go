package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
)

// memReader is an in-memory ActivityReader over a desc-sorted event slice, honoring
// the keyset cursor and limit exactly like the SQL repo.
type memReader struct{ events []entity.ActivityEvent }

func (m *memReader) ProfileEvents(_ context.Context, userID int64, cursor *repository.FeedCursor, limit int) ([]entity.ActivityEvent, error) {
	return m.page(func(e entity.ActivityEvent) bool { return e.UserID == userID }, cursor, limit), nil
}

func (m *memReader) HomeEvents(_ context.Context, userIDs []int64, cursor *repository.FeedCursor, limit int) ([]entity.ActivityEvent, error) {
	set := map[int64]bool{}
	for _, id := range userIDs {
		set[id] = true
	}
	return m.page(func(e entity.ActivityEvent) bool { return set[e.UserID] }, cursor, limit), nil
}

func (m *memReader) page(keep func(entity.ActivityEvent) bool, cursor *repository.FeedCursor, limit int) []entity.ActivityEvent {
	var out []entity.ActivityEvent
	for _, e := range m.events {
		if !keep(e) {
			continue
		}
		if cursor != nil && !before(e, *cursor) {
			continue
		}
		out = append(out, e)
		if len(out) == limit {
			break
		}
	}
	return out
}

// before reports whether e sorts strictly after the cursor in (created_at DESC, id DESC).
func before(e entity.ActivityEvent, c repository.FeedCursor) bool {
	if e.CreatedAt.Equal(c.CreatedAt) {
		return e.ID < c.ID
	}
	return e.CreatedAt.Before(c.CreatedAt)
}

type noFollowees struct{}

func (noFollowees) AcceptedFolloweeIDs(context.Context, int64) ([]int64, error) { return nil, nil }

func mkWatched(id, show int64, at time.Time) entity.ActivityEvent {
	ep := id
	season := 1
	return entity.ActivityEvent{ID: id, UserID: 1, Type: entity.EventWatchedEpisode, ShowID: show, SeasonNumber: &season, EpisodeID: &ep, CreatedAt: at}
}

func TestFeedUsecase_GroupingAndStandalones(t *testing.T) {
	T := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	nine := 9
	events := []entity.ActivityEvent{
		mkWatched(100, 1, T),
		mkWatched(99, 1, T.Add(-1*time.Hour)),
		mkWatched(98, 1, T.Add(-2*time.Hour)),
		// 18h gap > 12h window → a new group even though it's the same show.
		mkWatched(97, 1, T.Add(-20*time.Hour)),
		{ID: 96, UserID: 1, Type: entity.EventRatedShow, ShowID: 2, Rating: &nine, CreatedAt: T.Add(-21 * time.Hour)},
		{ID: 95, UserID: 1, Type: entity.EventFinishedShow, ShowID: 1, CreatedAt: T.Add(-22 * time.Hour)},
	}
	uc := usecase.NewFeedUsecase(&memReader{events: events}, noFollowees{}, 12*time.Hour)

	cards, next, err := uc.ProfileFeed(context.Background(), 1, "", 10)
	if err != nil {
		t.Fatalf("profile feed: %v", err)
	}
	if next != "" {
		t.Fatalf("want no next cursor on a short page, got %q", next)
	}
	if len(cards) != 4 {
		t.Fatalf("want 4 cards, got %d: %+v", len(cards), cards)
	}
	if cards[0].Type != entity.EventWatchedEpisode || cards[0].Count() != 3 {
		t.Fatalf("card0 should group 3 episodes, got %+v", cards[0])
	}
	if cards[1].Type != entity.EventWatchedEpisode || cards[1].Count() != 1 {
		t.Fatalf("card1 should be a lone watch past the window, got %+v", cards[1])
	}
	if cards[2].Type != entity.EventRatedShow || cards[2].Rating == nil || *cards[2].Rating != 9 {
		t.Fatalf("card2 should be rated_show(9), got %+v", cards[2])
	}
	if cards[3].Type != entity.EventFinishedShow {
		t.Fatalf("card3 should be finished_show, got %+v", cards[3])
	}
}

func TestFeedUsecase_KeysetPaginationStable(t *testing.T) {
	T := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	// Six standalone added_show events, 2 days apart so none group.
	var events []entity.ActivityEvent
	for i := 0; i < 6; i++ {
		id := int64(200 - i)
		events = append(events, entity.ActivityEvent{
			ID: id, UserID: 1, Type: entity.EventAddedShow, ShowID: int64(10 + i),
			CreatedAt: T.Add(time.Duration(-i) * 48 * time.Hour),
		})
	}
	reader := &memReader{events: events}
	uc := usecase.NewFeedUsecase(reader, noFollowees{}, 12*time.Hour)
	ctx := context.Background()

	page1, next, err := uc.ProfileFeed(ctx, 1, "", 2)
	if err != nil || len(page1) != 2 || next == "" {
		t.Fatalf("page1: len=%d next=%q err=%v", len(page1), next, err)
	}
	if page1[0].ShowID != 10 || page1[1].ShowID != 11 {
		t.Fatalf("page1 order wrong: %d, %d", page1[0].ShowID, page1[1].ShowID)
	}

	// Append a NEWER event before reading page 2; keyset must ignore it.
	reader.events = append([]entity.ActivityEvent{{
		ID: 999, UserID: 1, Type: entity.EventAddedShow, ShowID: 99, CreatedAt: T.Add(time.Hour),
	}}, reader.events...)

	page2, next2, err := uc.ProfileFeed(ctx, 1, next, 2)
	if err != nil || len(page2) != 2 {
		t.Fatalf("page2: len=%d err=%v", len(page2), err)
	}
	if page2[0].ShowID != 12 || page2[1].ShowID != 13 {
		t.Fatalf("page2 should continue past the cursor unaffected by new events: %d, %d", page2[0].ShowID, page2[1].ShowID)
	}
	if next2 == "" {
		t.Fatal("want a cursor for page3 (two events remain)")
	}
	for _, c := range page2 {
		if c.ShowID == 99 {
			t.Fatal("newer event leaked into a later page")
		}
	}
}

func TestFeedUsecase_BingeLongerThanOverfetch(t *testing.T) {
	T := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	// 35 consecutive watched episodes of one show, 10 min apart — a single binge that
	// exceeds the initial raw over-fetch (30 for limit=1), forcing the iterator to
	// fetch more to resolve the whole group into one full card.
	var events []entity.ActivityEvent
	for i := 0; i < 35; i++ {
		events = append(events, mkWatched(int64(1000-i), 1, T.Add(time.Duration(-i)*10*time.Minute)))
	}
	// A trailing standalone event so a second page exists.
	events = append(events, entity.ActivityEvent{
		ID: 900, UserID: 1, Type: entity.EventAddedShow, ShowID: 1, CreatedAt: T.Add(-48 * time.Hour),
	})
	uc := usecase.NewFeedUsecase(&memReader{events: events}, noFollowees{}, 12*time.Hour)
	ctx := context.Background()

	page1, next, err := uc.ProfileFeed(ctx, 1, "", 1)
	if err != nil || len(page1) != 1 {
		t.Fatalf("page1: len=%d err=%v", len(page1), err)
	}
	if page1[0].Count() != 35 {
		t.Fatalf("binge should fill one card with 35 episodes, got %d", page1[0].Count())
	}
	if next == "" {
		t.Fatal("want a cursor after the binge card")
	}
	page2, _, err := uc.ProfileFeed(ctx, 1, next, 1)
	if err != nil || len(page2) != 1 || page2[0].Type != entity.EventAddedShow {
		t.Fatalf("page2 should be the trailing added_show: %+v err=%v", page2, err)
	}
}

func TestFeedUsecase_HomeEmptyWhenNoFollowees(t *testing.T) {
	uc := usecase.NewFeedUsecase(&memReader{}, noFollowees{}, 12*time.Hour)
	cards, next, err := uc.HomeFeed(context.Background(), 1, "", 10)
	if err != nil || len(cards) != 0 || next != "" {
		t.Fatalf("home feed with no followees should be empty: len=%d next=%q err=%v", len(cards), next, err)
	}
}
