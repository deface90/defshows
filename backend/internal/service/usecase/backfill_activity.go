package usecase

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/deface90/defshows/backend/internal/service/entity"
)

// ActivityBackfill seeds the activity_events feed from pre-existing tracking state
// (shows added, episodes watched) so the feed has history from before emission was
// woven into the tracking transactions. It is idempotent: an event already present
// (matched on its natural key) is never re-created, so the command is safe to re-run.
//
// Note: historical bulk WatchShow runs share watched_at = now() across many episodes,
// so backfilled watched/finished events may bunch at a single timestamp. That is
// accepted — the original per-episode order is unrecoverable.
type ActivityBackfill struct {
	db *gorm.DB
}

// NewActivityBackfill creates an ActivityBackfill over the given database.
func NewActivityBackfill(db *gorm.DB) *ActivityBackfill {
	return &ActivityBackfill{db: db}
}

type showKey struct{ user, show int64 }
type episodeKey struct{ user, episode int64 }
type seasonKey struct {
	user, show int64
	season     int
}

// Run generates the missing activity events in one batch and returns how many it
// inserted. Zero means the feed was already complete (a no-op re-run).
func (b *ActivityBackfill) Run(ctx context.Context) (int, error) {
	db := b.db.WithContext(ctx)

	var userShows []entity.UserShow
	if err := db.Find(&userShows).Error; err != nil {
		return 0, err
	}
	if len(userShows) == 0 {
		return 0, nil
	}
	type usRef struct {
		userID, showID int64
	}
	usByID := make(map[int64]usRef, len(userShows))
	showIDset := make(map[int64]struct{})
	for _, us := range userShows {
		usByID[us.ID] = usRef{us.UserID, us.ShowID}
		showIDset[us.ShowID] = struct{}{}
	}

	showIDs := make([]int64, 0, len(showIDset))
	for id := range showIDset {
		showIDs = append(showIDs, id)
	}
	var episodes []entity.Episode
	if err := db.Where("show_id IN ?", showIDs).Find(&episodes).Error; err != nil {
		return 0, err
	}
	epByID := make(map[int64]*entity.Episode, len(episodes))
	epsByShow := make(map[int64][]*entity.Episode)
	for i := range episodes {
		e := &episodes[i]
		epByID[e.ID] = e
		epsByShow[e.ShowID] = append(epsByShow[e.ShowID], e)
	}

	var marks []entity.UserEpisode
	if err := db.Where("watched = true").Find(&marks).Error; err != nil {
		return 0, err
	}

	added, watched, finishedSeason, finishedShow, err := b.loadExisting(db)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	var out []entity.ActivityEvent

	// added_show ← user_shows.added_at
	for _, us := range userShows {
		k := showKey{us.UserID, us.ShowID}
		if added[k] {
			continue
		}
		out = append(out, entity.ActivityEvent{
			UserID: us.UserID, Type: entity.EventAddedShow, ShowID: us.ShowID, CreatedAt: us.AddedAt,
		})
		added[k] = true
	}

	// watched_episode ← user_episodes.watched_at; also accumulate watched sets per
	// user_show so season/show completion can be derived below.
	watchedByUS := map[int64]map[int64]time.Time{}
	for i := range marks {
		m := &marks[i]
		us, ok := usByID[m.UserShowID]
		if !ok {
			continue
		}
		ep := epByID[m.EpisodeID]
		if ep == nil {
			continue
		}
		at := now
		if m.WatchedAt != nil {
			at = *m.WatchedAt
		}
		if watchedByUS[m.UserShowID] == nil {
			watchedByUS[m.UserShowID] = map[int64]time.Time{}
		}
		watchedByUS[m.UserShowID][m.EpisodeID] = at

		ek := episodeKey{us.userID, m.EpisodeID}
		if watched[ek] {
			continue
		}
		season := ep.SeasonNumber
		episodeID := m.EpisodeID
		out = append(out, entity.ActivityEvent{
			UserID: us.userID, Type: entity.EventWatchedEpisode, ShowID: us.showID,
			SeasonNumber: &season, EpisodeID: &episodeID, CreatedAt: at,
		})
		watched[ek] = true
	}

	// finished_season / finished_show computed from the watched sets over aired
	// episodes; created_at = the latest watch in that season / show.
	for usID, w := range watchedByUS {
		us := usByID[usID]
		type tally struct {
			aired, done int
			maxAt       time.Time
		}
		seasons := map[int]*tally{}
		showAired, showDone := 0, 0
		var showMaxAt time.Time
		for _, e := range epsByShow[us.showID] {
			if !aired(e.AirDate, now) {
				continue
			}
			at, done := w[e.ID]
			showAired++
			if done {
				showDone++
				if at.After(showMaxAt) {
					showMaxAt = at
				}
			}
			if e.SeasonNumber > 0 {
				st := seasons[e.SeasonNumber]
				if st == nil {
					st = &tally{}
					seasons[e.SeasonNumber] = st
				}
				st.aired++
				if done {
					st.done++
					if at.After(st.maxAt) {
						st.maxAt = at
					}
				}
			}
		}
		for sn, st := range seasons {
			if st.aired == 0 || st.done != st.aired {
				continue
			}
			k := seasonKey{us.userID, us.showID, sn}
			if finishedSeason[k] {
				continue
			}
			n := sn
			out = append(out, entity.ActivityEvent{
				UserID: us.userID, Type: entity.EventFinishedSeason, ShowID: us.showID,
				SeasonNumber: &n, CreatedAt: nonZero(st.maxAt, now),
			})
			finishedSeason[k] = true
		}
		if showAired > 0 && showDone == showAired {
			k := showKey{us.userID, us.showID}
			if !finishedShow[k] {
				out = append(out, entity.ActivityEvent{
					UserID: us.userID, Type: entity.EventFinishedShow, ShowID: us.showID,
					CreatedAt: nonZero(showMaxAt, now),
				})
				finishedShow[k] = true
			}
		}
	}

	if len(out) == 0 {
		return 0, nil
	}
	if err := db.CreateInBatches(&out, 500).Error; err != nil {
		return 0, err
	}
	return len(out), nil
}

func (b *ActivityBackfill) loadExisting(db *gorm.DB) (
	added map[showKey]bool, watched map[episodeKey]bool,
	finishedSeason map[seasonKey]bool, finishedShow map[showKey]bool, err error,
) {
	added = map[showKey]bool{}
	watched = map[episodeKey]bool{}
	finishedSeason = map[seasonKey]bool{}
	finishedShow = map[showKey]bool{}
	var events []entity.ActivityEvent
	if err = db.Find(&events).Error; err != nil {
		return
	}
	for i := range events {
		e := &events[i]
		switch e.Type {
		case entity.EventAddedShow:
			added[showKey{e.UserID, e.ShowID}] = true
		case entity.EventWatchedEpisode:
			if e.EpisodeID != nil {
				watched[episodeKey{e.UserID, *e.EpisodeID}] = true
			}
		case entity.EventFinishedSeason:
			if e.SeasonNumber != nil {
				finishedSeason[seasonKey{e.UserID, e.ShowID, *e.SeasonNumber}] = true
			}
		case entity.EventFinishedShow:
			finishedShow[showKey{e.UserID, e.ShowID}] = true
		}
	}
	return
}

func nonZero(t, fallback time.Time) time.Time {
	if t.IsZero() {
		return fallback
	}
	return t
}
