package usecase

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
)

// Tracking usecase errors.
var (
	ErrAlreadyTracked = errors.New("usecase: show already tracked")
	ErrNotTracked     = errors.New("usecase: show not tracked")
	ErrInvalidRating  = errors.New("usecase: rating must be between 1 and 10")
)

// TrackingRepo is the storage dependency of TrackingUsecase.
type TrackingRepo interface {
	UpdateLink(ctx context.Context, userShowID, linkID int64, value string) error
	WatchShow(ctx context.Context, userID, showID int64, events ...entity.ActivityEvent) error
	AddUserShow(ctx context.Context, us *entity.UserShow, events ...entity.ActivityEvent) error
	GetUserShow(ctx context.Context, userID, showID int64) (*entity.UserShow, error)
	ListUserShows(ctx context.Context, userID int64, status string) ([]entity.UserShow, error)
	UpdateUserShow(ctx context.Context, us *entity.UserShow) error
	SetRating(ctx context.Context, userShowID int64, rating *int, events ...entity.ActivityEvent) error
	RemoveUserShow(ctx context.Context, userID, showID int64) error
	UpsertUserEpisode(ctx context.Context, ue *entity.UserEpisode, events ...entity.ActivityEvent) error
	WatchedEpisodeIDs(ctx context.Context, userShowID int64) ([]int64, error)
	AddLink(ctx context.Context, link *entity.UserShowLink) error
	ListLinks(ctx context.Context, userShowID int64) ([]entity.UserShowLink, error)
	DeleteLink(ctx context.Context, userShowID, linkID int64) error
	CountByUsers(ctx context.Context, userIDs ...int64) (map[int64]int, error)
}

// Catalog is the catalog dependency of TrackingUsecase.
type Catalog interface {
	EnsureShow(ctx context.Context, tmdbID int64) (*entity.Show, error)
	ShowByID(ctx context.Context, showID int64) (*entity.Show, error)
	Episodes(ctx context.Context, showID int64) ([]entity.Episode, error)
}

// Progress summarizes viewing of aired episodes only.
// WatchedEpisodeIDs preserves all saved marks, including dates later changed by the provider.
type Progress struct {
	WatchedEpisodeIDs []int64
	Watched           int
	Total             int
	// Unwatched counts episodes that have already aired but are not marked
	// watched — i.e. what the user can actually watch right now.
	Unwatched     int
	NextUnwatched *entity.Episode
	// NewFullSeason is the number of a fully-aired, completely-unwatched season
	// that dropped while the user was caught up on everything earlier; nil when
	// there is no such season.
	NewFullSeason *int
}

// Stats aggregates a user's viewing across all tracked shows.
type Stats struct {
	ShowsTracked    int
	ShowsCompleted  int
	SeasonsWatched  int
	EpisodesWatched int
	// MinutesWatched sums the runtime of watched episodes (episodes with an
	// unknown runtime contribute zero).
	MinutesWatched int64
}

// TrackedShow bundles a user's tracking with the show and its progress.
type TrackedShow struct {
	UserShow *entity.UserShow
	Show     *entity.Show
	Progress Progress
}

// TrackingUsecase implements the user tracking workflow.
type TrackingUsecase struct {
	repo    TrackingRepo
	catalog Catalog
}

// NewTrackingUsecase creates a TrackingUsecase.
func NewTrackingUsecase(repo TrackingRepo, catalog Catalog) *TrackingUsecase {
	return &TrackingUsecase{repo: repo, catalog: catalog}
}

// AddShow adds a show to the user's list, importing it from the provider on
// first use.
func (uc *TrackingUsecase) AddShow(ctx context.Context, userID, tmdbID int64) (*entity.UserShow, error) {
	show, err := uc.catalog.EnsureShow(ctx, tmdbID)
	if err != nil {
		return nil, err
	}
	us := &entity.UserShow{UserID: userID, ShowID: show.ID, Status: entity.StatusWatching}
	event := entity.ActivityEvent{UserID: userID, Type: entity.EventAddedShow, ShowID: show.ID}
	if err := uc.repo.AddUserShow(ctx, us, event); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrAlreadyTracked
		}
		return nil, err
	}
	return us, nil
}

// ShowsCountByUser returns the number of tracked shows per user id.
func (uc *TrackingUsecase) ShowsCountByUser(ctx context.Context, userIDs ...int64) (map[int64]int, error) {
	return uc.repo.CountByUsers(ctx, userIDs...)
}

// RemoveShow removes a show from the user's list.
func (uc *TrackingUsecase) RemoveShow(ctx context.Context, userID, showID int64) error {
	return uc.repo.RemoveUserShow(ctx, userID, showID)
}

// ListShows returns the user's tracked shows with progress.
func (uc *TrackingUsecase) ListShows(ctx context.Context, userID int64, status string) ([]TrackedShow, error) {
	userShows, err := uc.repo.ListUserShows(ctx, userID, status)
	if err != nil {
		return nil, err
	}
	out := make([]TrackedShow, 0, len(userShows))
	for i := range userShows {
		us := userShows[i]
		ts, err := uc.tracked(ctx, &us)
		if err != nil {
			return nil, err
		}
		out = append(out, ts)
	}
	return out, nil
}

// GetTracked returns one tracked show with its show info and progress.
func (uc *TrackingUsecase) GetTracked(ctx context.Context, userID, showID int64) (*TrackedShow, error) {
	us, err := uc.repo.GetUserShow(ctx, userID, showID)
	if err != nil {
		return nil, notTracked(err)
	}
	ts, err := uc.tracked(ctx, us)
	if err != nil {
		return nil, err
	}
	return &ts, nil
}

func (uc *TrackingUsecase) tracked(ctx context.Context, us *entity.UserShow) (TrackedShow, error) {
	prog, err := uc.progress(ctx, us)
	if err != nil {
		return TrackedShow{}, err
	}
	show, err := uc.catalog.ShowByID(ctx, us.ShowID)
	if err != nil {
		return TrackedShow{}, err
	}
	return TrackedShow{UserShow: us, Show: show, Progress: prog}, nil
}

// UpdateShow updates status/favorite/preferred dubbing.
// UpdateShow applies a partial update. status/favorite/dubbing follow absent=unchanged.
// Rating is special: rating!=nil sets it (validated 1..10), while clearRating removes it;
// if neither is given the rating is left untouched. Rating is persisted separately from
// the other fields so the two never clobber each other.
func (uc *TrackingUsecase) UpdateShow(ctx context.Context, userID, showID int64, status *string, favorite *bool, dubbing *string, rating *int, clearRating bool) (*entity.UserShow, error) {
	if rating != nil && (*rating < 1 || *rating > 10) {
		return nil, ErrInvalidRating
	}
	us, err := uc.repo.GetUserShow(ctx, userID, showID)
	if err != nil {
		return nil, notTracked(err)
	}
	if status != nil {
		us.Status = entity.WatchStatus(*status)
	}
	if favorite != nil {
		us.Favorite = *favorite
	}
	if dubbing != nil {
		us.PreferredDubbing = *dubbing
	}
	if err := uc.repo.UpdateUserShow(ctx, us); err != nil {
		return nil, err
	}
	if rating != nil || clearRating {
		newRating := rating // nil when clearRating
		var events []entity.ActivityEvent
		// Emit rated_show on a genuine set or change, but not on a clear or a no-op
		// re-set of the same value.
		if rating != nil && (us.Rating == nil || *us.Rating != *rating) {
			events = append(events, entity.ActivityEvent{
				UserID: userID, Type: entity.EventRatedShow, ShowID: showID, Rating: rating,
			})
		}
		if err := uc.repo.SetRating(ctx, us.ID, newRating, events...); err != nil {
			return nil, err
		}
		us.Rating = newRating
	}
	return us, nil
}

// SetEpisodeWatched marks or unmarks an episode as watched. Marking a newly-watched
// episode emits a watched_episode event, plus finished_season / finished_show when
// that mark completes the season or the whole show (aired episodes only). Un-marking,
// or re-marking an already-watched episode, emits nothing.
func (uc *TrackingUsecase) SetEpisodeWatched(ctx context.Context, userID, showID, episodeID int64, watched bool) error {
	us, err := uc.repo.GetUserShow(ctx, userID, showID)
	if err != nil {
		return notTracked(err)
	}
	ue := &entity.UserEpisode{UserShowID: us.ID, EpisodeID: episodeID, Watched: watched}
	var events []entity.ActivityEvent
	if watched {
		now := time.Now()
		ue.WatchedAt = &now
		if events, err = uc.watchEpisodeEvents(ctx, us, episodeID, now); err != nil {
			return err
		}
	}
	return uc.repo.UpsertUserEpisode(ctx, ue, events...)
}

// watchEpisodeEvents builds the activity events produced by marking episodeID
// watched: a watched_episode event, and finished_season / finished_show when this
// mark completes the season or the show (over aired episodes). Returns no events when
// the episode was already watched (idempotent re-mark) or is unknown to the catalog.
func (uc *TrackingUsecase) watchEpisodeEvents(ctx context.Context, us *entity.UserShow, episodeID int64, now time.Time) ([]entity.ActivityEvent, error) {
	episodes, err := uc.catalog.Episodes(ctx, us.ShowID)
	if err != nil {
		return nil, err
	}
	watchedIDs, err := uc.repo.WatchedEpisodeIDs(ctx, us.ID)
	if err != nil {
		return nil, err
	}
	watched := make(map[int64]bool, len(watchedIDs))
	for _, id := range watchedIDs {
		watched[id] = true
	}
	if watched[episodeID] {
		return nil, nil // already watched — the mark is a no-op, emit nothing
	}
	var target *entity.Episode
	for i := range episodes {
		if episodes[i].ID == episodeID {
			target = &episodes[i]
			break
		}
	}
	if target == nil {
		return nil, nil // not in the catalog — nothing to describe
	}
	season := target.SeasonNumber
	events := []entity.ActivityEvent{{
		UserID: us.UserID, Type: entity.EventWatchedEpisode, ShowID: us.ShowID,
		SeasonNumber: &season, EpisodeID: &episodeID,
	}}
	// Completion is defined over aired episodes only, matching progress semantics.
	// An unaired mark can never complete anything, so it stays a lone watched event.
	if !aired(target.AirDate, now) {
		return events, nil
	}
	seasonAired, seasonWatched := 0, 0
	showAired, showWatched := 0, 0
	for i := range episodes {
		e := &episodes[i]
		if !aired(e.AirDate, now) {
			continue
		}
		watchedAfter := watched[e.ID] || e.ID == episodeID
		showAired++
		if watchedAfter {
			showWatched++
		}
		if e.SeasonNumber == season {
			seasonAired++
			if watchedAfter {
				seasonWatched++
			}
		}
	}
	if season > 0 && seasonAired > 0 && seasonWatched == seasonAired {
		sn := season
		events = append(events, entity.ActivityEvent{
			UserID: us.UserID, Type: entity.EventFinishedSeason, ShowID: us.ShowID, SeasonNumber: &sn,
		})
	}
	if showAired > 0 && showWatched == showAired {
		events = append(events, entity.ActivityEvent{
			UserID: us.UserID, Type: entity.EventFinishedShow, ShowID: us.ShowID,
		})
	}
	return events, nil
}

// GetProgress returns progress for a tracked show.
func (uc *TrackingUsecase) GetProgress(ctx context.Context, userID, showID int64) (Progress, error) {
	us, err := uc.repo.GetUserShow(ctx, userID, showID)
	if err != nil {
		return Progress{}, notTracked(err)
	}
	return uc.progress(ctx, us)
}

// AddLink adds a reference link to a tracked show.
func (uc *TrackingUsecase) AddLink(ctx context.Context, userID, showID int64, kind, label, url string) (*entity.UserShowLink, error) {
	us, err := uc.repo.GetUserShow(ctx, userID, showID)
	if err != nil {
		return nil, notTracked(err)
	}
	link := &entity.UserShowLink{UserShowID: us.ID, Kind: entity.LinkKind(kind), Label: label, URL: url}
	if err := uc.repo.AddLink(ctx, link); err != nil {
		return nil, err
	}
	return link, nil
}

// ListLinks lists a tracked show's links.
func (uc *TrackingUsecase) ListLinks(ctx context.Context, userID, showID int64) ([]entity.UserShowLink, error) {
	us, err := uc.repo.GetUserShow(ctx, userID, showID)
	if err != nil {
		return nil, notTracked(err)
	}
	return uc.repo.ListLinks(ctx, us.ID)
}

// DeleteLink removes a link from a tracked show.
func (uc *TrackingUsecase) DeleteLink(ctx context.Context, userID, showID, linkID int64) error {
	us, err := uc.repo.GetUserShow(ctx, userID, showID)
	if err != nil {
		return notTracked(err)
	}
	return uc.repo.DeleteLink(ctx, us.ID, linkID)
}

func (uc *TrackingUsecase) progress(ctx context.Context, us *entity.UserShow) (Progress, error) {
	episodes, err := uc.catalog.Episodes(ctx, us.ShowID)
	if err != nil {
		return Progress{}, err
	}
	watchedIDs, err := uc.repo.WatchedEpisodeIDs(ctx, us.ID)
	if err != nil {
		return Progress{}, err
	}
	watched := make(map[int64]bool, len(watchedIDs))
	for _, id := range watchedIDs {
		watched[id] = true
	}

	prog := Progress{WatchedEpisodeIDs: append([]int64{}, watchedIDs...)}
	now := time.Now()
	seasons := map[int]*seasonTally{}
	for i := range episodes {
		e := &episodes[i]
		if e.SeasonNumber > 0 {
			st := seasons[e.SeasonNumber]
			if st == nil {
				st = &seasonTally{}
				seasons[e.SeasonNumber] = st
			}
			st.total++
			if aired(e.AirDate, now) {
				st.aired++
				if watched[e.ID] {
					st.watched++
				}
			}
		}
		if !aired(e.AirDate, now) {
			continue
		}
		prog.Total++
		if watched[e.ID] {
			prog.Watched++
			continue
		}
		prog.Unwatched++
		if prog.NextUnwatched == nil {
			prog.NextUnwatched = e
		}
	}
	prog.NewFullSeason = newFullSeason(seasons)
	return prog, nil
}

// seasonTally counts a single season's episodes: catalogued (total), already
// aired, and watched-among-aired.
type seasonTally struct{ total, aired, watched int }

// newFullSeason returns the earliest season that has fully aired with zero
// watches while every earlier season is fully caught up — i.e. a whole new
// season dropped for a user who was up to date. Returns nil otherwise.
func newFullSeason(seasons map[int]*seasonTally) *int {
	nums := make([]int, 0, len(seasons))
	for n := range seasons {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	for _, n := range nums {
		st := seasons[n]
		if st.aired > 0 && st.aired == st.total && st.watched == 0 {
			num := n
			return &num
		}
		// An earlier season with aired-but-unwatched episodes means the user is
		// mid-run, not waiting on a whole new season.
		if st.watched < st.aired {
			return nil
		}
	}
	return nil
}

// Stats aggregates the user's viewing across all their tracked shows.
func (uc *TrackingUsecase) Stats(ctx context.Context, userID int64) (Stats, error) {
	userShows, err := uc.repo.ListUserShows(ctx, userID, "")
	if err != nil {
		return Stats{}, err
	}
	now := time.Now()
	st := Stats{ShowsTracked: len(userShows)}
	for i := range userShows {
		us := userShows[i]
		if us.Status == entity.StatusCompleted {
			st.ShowsCompleted++
		}
		episodes, err := uc.catalog.Episodes(ctx, us.ShowID)
		if err != nil {
			return Stats{}, err
		}
		watchedIDs, err := uc.repo.WatchedEpisodeIDs(ctx, us.ID)
		if err != nil {
			return Stats{}, err
		}
		watched := make(map[int64]bool, len(watchedIDs))
		for _, id := range watchedIDs {
			watched[id] = true
		}
		airedBySeason := map[int]int{}
		watchedBySeason := map[int]int{}
		for j := range episodes {
			e := &episodes[j]
			if e.SeasonNumber <= 0 {
				continue
			}
			if aired(e.AirDate, now) {
				airedBySeason[e.SeasonNumber]++
			}
			if watched[e.ID] {
				st.EpisodesWatched++
				watchedBySeason[e.SeasonNumber]++
				if e.Runtime != nil {
					st.MinutesWatched += int64(*e.Runtime)
				}
			}
		}
		for sn, airedCnt := range airedBySeason {
			if airedCnt > 0 && watchedBySeason[sn] >= airedCnt {
				st.SeasonsWatched++
			}
		}
	}
	return st, nil
}

// aired reports whether an episode with the given air date has already been
// released (nil air date = unknown, treated as not yet aired).
func aired(airDate *time.Time, now time.Time) bool {
	return airDate != nil && !airDate.After(now)
}

func notTracked(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotTracked
	}
	return err
}

// WatchShow marks all currently-aired episodes watched, without changing the
// show's status. Because every aired episode ends up watched, it emits a single
// finished_show event plus one finished_season per aired season — never per-episode
// watched events. Emits nothing when the show has no aired episodes.
func (uc *TrackingUsecase) WatchShow(ctx context.Context, userID, showID int64) error {
	episodes, err := uc.catalog.Episodes(ctx, showID)
	if err != nil {
		return err
	}
	now := time.Now()
	airedSeasons := map[int]bool{}
	anyAired := false
	for i := range episodes {
		e := &episodes[i]
		if !aired(e.AirDate, now) {
			continue
		}
		anyAired = true
		if e.SeasonNumber > 0 {
			airedSeasons[e.SeasonNumber] = true
		}
	}
	var events []entity.ActivityEvent
	if anyAired {
		nums := make([]int, 0, len(airedSeasons))
		for n := range airedSeasons {
			nums = append(nums, n)
		}
		sort.Ints(nums)
		for _, n := range nums {
			sn := n
			events = append(events, entity.ActivityEvent{
				UserID: userID, Type: entity.EventFinishedSeason, ShowID: showID, SeasonNumber: &sn,
			})
		}
		events = append(events, entity.ActivityEvent{
			UserID: userID, Type: entity.EventFinishedShow, ShowID: showID,
		})
	}
	return notTracked(uc.repo.WatchShow(ctx, userID, showID, events...))
}

// UpdateLink edits an existing link without changing its identity, label or kind.
func (uc *TrackingUsecase) UpdateLink(ctx context.Context, userID, showID, linkID int64, value string) error {
	us, err := uc.repo.GetUserShow(ctx, userID, showID)
	if err != nil {
		return notTracked(err)
	}
	return notTracked(uc.repo.UpdateLink(ctx, us.ID, linkID, value))
}
