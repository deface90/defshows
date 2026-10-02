package usecase

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
)

const (
	defaultFeedLimit = 20
	maxFeedLimit     = 50
)

// ActivityReader reads raw activity events in keyset pages (newest first).
type ActivityReader interface {
	ProfileEvents(ctx context.Context, userID int64, cursor *repository.FeedCursor, limit int) ([]entity.ActivityEvent, error)
	HomeEvents(ctx context.Context, userIDs []int64, cursor *repository.FeedCursor, limit int) ([]entity.ActivityEvent, error)
}

// FolloweeLister yields the accepted-followee set backing the home feed.
type FolloweeLister interface {
	AcceptedFolloweeIDs(ctx context.Context, userID int64) ([]int64, error)
}

// FeedItem is one rendered feed card. Consecutive watched_episode events of the same
// (actor, show) within the group window collapse into a single item whose EpisodeIDs
// holds every episode watched, newest-first; the other event types stand alone.
type FeedItem struct {
	ActorID    int64
	Type       entity.EventType
	ShowID     int64
	Season     *int    // finished_season
	Rating     *int    // rated_show
	EpisodeIDs []int64 // watched_episode (len 1 for a single watch)
	CreatedAt  time.Time

	// oldest raw event folded into this card, for keyset continuation.
	oldestAt time.Time
	oldestID int64
}

// Count is the number of episodes a watched_episode card groups (1 for a single).
func (i FeedItem) Count() int { return len(i.EpisodeIDs) }

// FeedUsecase builds the profile and home activity feeds with read-time grouping and
// keyset pagination.
type FeedUsecase struct {
	events ActivityReader
	social FolloweeLister
	window time.Duration
}

// NewFeedUsecase creates a FeedUsecase. window is the max gap between consecutive
// watched episodes that still collapse into one card.
func NewFeedUsecase(events ActivityReader, social FolloweeLister, window time.Duration) *FeedUsecase {
	return &FeedUsecase{events: events, social: social, window: window}
}

// ProfileFeed returns one page of a single user's activity.
func (uc *FeedUsecase) ProfileFeed(ctx context.Context, userID int64, cursor string, limit int) ([]FeedItem, string, error) {
	cur, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	limit = clampFeedLimit(limit)
	fetch := func(c *repository.FeedCursor, n int) ([]entity.ActivityEvent, error) {
		return uc.events.ProfileEvents(ctx, userID, c, n)
	}
	return uc.page(fetch, cur, limit)
}

// HomeFeed returns one page of the aggregated activity of everyone the viewer follows
// (accepted). A viewer who follows nobody gets an empty feed.
func (uc *FeedUsecase) HomeFeed(ctx context.Context, viewerID int64, cursor string, limit int) ([]FeedItem, string, error) {
	followees, err := uc.social.AcceptedFolloweeIDs(ctx, viewerID)
	if err != nil {
		return nil, "", err
	}
	if len(followees) == 0 {
		return []FeedItem{}, "", nil
	}
	cur, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	limit = clampFeedLimit(limit)
	fetch := func(c *repository.FeedCursor, n int) ([]entity.ActivityEvent, error) {
		return uc.events.HomeEvents(ctx, followees, c, n)
	}
	return uc.page(fetch, cur, limit)
}

type fetchFn func(cursor *repository.FeedCursor, limit int) ([]entity.ActivityEvent, error)

// page groups a raw event stream into up to `limit` cards and returns the next
// cursor. It keeps pulling raw rows (over-fetching, and fetching more when a long
// binge would under-fill the page) until the page is full or the stream is exhausted.
func (uc *FeedUsecase) page(fetch fetchFn, cursor *repository.FeedCursor, limit int) ([]FeedItem, string, error) {
	it := &eventIter{fetch: fetch, batch: rawBatch(limit), cursor: cursor}

	cards := make([]FeedItem, 0, limit)
	var open *FeedItem
	var openKey groupKey
	reachedLimit := false

	for {
		ev, ok, err := it.next()
		if err != nil {
			return nil, "", err
		}
		if !ok {
			break // stream exhausted — flush any open group below
		}
		if ev.Type == entity.EventWatchedEpisode {
			k := groupKey{actor: ev.UserID, show: ev.ShowID}
			if open != nil && k == openKey && open.oldestAt.Sub(ev.CreatedAt) <= uc.window {
				open.EpisodeIDs = append(open.EpisodeIDs, deref(ev.EpisodeID))
				open.oldestAt, open.oldestID = ev.CreatedAt, ev.ID
				continue
			}
			if open != nil {
				cards = append(cards, *open)
				open = nil
				if len(cards) == limit {
					reachedLimit = true
					break
				}
			}
			open = &FeedItem{
				ActorID: ev.UserID, Type: entity.EventWatchedEpisode, ShowID: ev.ShowID,
				Season: ev.SeasonNumber, EpisodeIDs: []int64{deref(ev.EpisodeID)},
				CreatedAt: ev.CreatedAt, oldestAt: ev.CreatedAt, oldestID: ev.ID,
			}
			openKey = k
			continue
		}
		// A standalone event closes any open watched group first.
		if open != nil {
			cards = append(cards, *open)
			open = nil
			if len(cards) == limit {
				reachedLimit = true
				break
			}
		}
		cards = append(cards, FeedItem{
			ActorID: ev.UserID, Type: ev.Type, ShowID: ev.ShowID, Season: ev.SeasonNumber,
			Rating: ev.Rating, CreatedAt: ev.CreatedAt, oldestAt: ev.CreatedAt, oldestID: ev.ID,
		})
		if len(cards) == limit {
			reachedLimit = true
			break
		}
	}
	if !reachedLimit && open != nil {
		cards = append(cards, *open)
	}

	next := ""
	if reachedLimit && len(cards) > 0 {
		last := cards[len(cards)-1]
		next = encodeCursor(&repository.FeedCursor{CreatedAt: last.oldestAt, ID: last.oldestID})
	}
	return cards, next, nil
}

type groupKey struct{ actor, show int64 }

// eventIter streams raw events, transparently fetching the next keyset page when its
// buffer drains, so a group that spans page boundaries can still be fully resolved.
type eventIter struct {
	fetch  fetchFn
	batch  int
	cursor *repository.FeedCursor
	buf    []entity.ActivityEvent
	pos    int
	done   bool
}

func (it *eventIter) next() (*entity.ActivityEvent, bool, error) {
	if it.pos >= len(it.buf) {
		if it.done {
			return nil, false, nil
		}
		rows, err := it.fetch(it.cursor, it.batch)
		if err != nil {
			return nil, false, err
		}
		if len(rows) < it.batch {
			it.done = true
		}
		if len(rows) == 0 {
			return nil, false, nil
		}
		it.buf, it.pos = rows, 0
		last := rows[len(rows)-1]
		it.cursor = &repository.FeedCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	ev := &it.buf[it.pos]
	it.pos++
	return ev, true, nil
}

func clampFeedLimit(limit int) int {
	if limit <= 0 {
		return defaultFeedLimit
	}
	if limit > maxFeedLimit {
		return maxFeedLimit
	}
	return limit
}

// rawBatch over-fetches raw rows per page so grouping rarely needs a second fetch.
func rawBatch(limit int) int {
	b := limit * 3
	if b < 30 {
		b = 30
	}
	return b
}

func deref(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// encodeCursor serializes a keyset position as opaque base64 of
// "<created_at RFC3339Nano>|<id>". Empty position encodes to "".
func encodeCursor(c *repository.FeedCursor) string {
	if c == nil {
		return ""
	}
	raw := fmt.Sprintf("%s|%d", c.CreatedAt.UTC().Format(time.RFC3339Nano), c.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeCursor parses a cursor produced by encodeCursor. Empty string = first page.
func decodeCursor(s string) (*repository.FeedCursor, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("feed: invalid cursor: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("feed: malformed cursor")
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, fmt.Errorf("feed: invalid cursor time: %w", err)
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("feed: invalid cursor id: %w", err)
	}
	return &repository.FeedCursor{CreatedAt: at, ID: id}, nil
}
