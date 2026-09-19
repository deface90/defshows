package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
)

// Tracking usecase errors.
var (
	ErrAlreadyTracked = errors.New("usecase: show already tracked")
	ErrNotTracked     = errors.New("usecase: show not tracked")
)

// TrackingRepo is the storage dependency of TrackingUsecase.
type TrackingRepo interface {
	UpdateLink(ctx context.Context, userShowID, linkID int64, value string) error
	WatchShow(ctx context.Context, userID, showID int64) error
	AddUserShow(ctx context.Context, us *entity.UserShow) error
	GetUserShow(ctx context.Context, userID, showID int64) (*entity.UserShow, error)
	ListUserShows(ctx context.Context, userID int64, status string) ([]entity.UserShow, error)
	UpdateUserShow(ctx context.Context, us *entity.UserShow) error
	RemoveUserShow(ctx context.Context, userID, showID int64) error
	UpsertUserEpisode(ctx context.Context, ue *entity.UserEpisode) error
	WatchedEpisodeIDs(ctx context.Context, userShowID int64) ([]int64, error)
	AddLink(ctx context.Context, link *entity.UserShowLink) error
	ListLinks(ctx context.Context, userShowID int64) ([]entity.UserShowLink, error)
	DeleteLink(ctx context.Context, userShowID, linkID int64) error
	CountByUsers(ctx context.Context) (map[int64]int, error)
}

// Catalog is the catalog dependency of TrackingUsecase.
type Catalog interface {
	EnsureShow(ctx context.Context, tmdbID int64) (*entity.Show, error)
	ShowByID(ctx context.Context, showID int64) (*entity.Show, error)
	Episodes(ctx context.Context, showID int64) ([]entity.Episode, error)
}

// Progress summarizes how far a user is through a show.
type Progress struct {
	WatchedEpisodeIDs []int64
	Watched           int
	Total             int
	// Unwatched counts episodes that have already aired but are not marked
	// watched — i.e. what the user can actually watch right now.
	Unwatched     int
	NextUnwatched *entity.Episode
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
	if err := uc.repo.AddUserShow(ctx, us); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrAlreadyTracked
		}
		return nil, err
	}
	return us, nil
}

// ShowsCountByUser returns the number of tracked shows per user id.
func (uc *TrackingUsecase) ShowsCountByUser(ctx context.Context) (map[int64]int, error) {
	return uc.repo.CountByUsers(ctx)
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
func (uc *TrackingUsecase) UpdateShow(ctx context.Context, userID, showID int64, status *string, favorite *bool, dubbing *string) (*entity.UserShow, error) {
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
	return us, nil
}

// SetEpisodeWatched marks or unmarks an episode as watched.
func (uc *TrackingUsecase) SetEpisodeWatched(ctx context.Context, userID, showID, episodeID int64, watched bool) error {
	us, err := uc.repo.GetUserShow(ctx, userID, showID)
	if err != nil {
		return notTracked(err)
	}
	ue := &entity.UserEpisode{UserShowID: us.ID, EpisodeID: episodeID, Watched: watched}
	if watched {
		now := time.Now()
		ue.WatchedAt = &now
	}
	return uc.repo.UpsertUserEpisode(ctx, ue)
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

	prog := Progress{Total: len(episodes), Watched: len(watchedIDs), WatchedEpisodeIDs: append([]int64{}, watchedIDs...)}
	now := time.Now()
	for i := range episodes {
		if watched[episodes[i].ID] {
			continue
		}
		if prog.NextUnwatched == nil {
			prog.NextUnwatched = &episodes[i]
		}
		if aired(episodes[i].AirDate, now) {
			prog.Unwatched++
		}
	}
	return prog, nil
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
// show's status.
func (uc *TrackingUsecase) WatchShow(ctx context.Context, userID, showID int64) error {
	return notTracked(uc.repo.WatchShow(ctx, userID, showID))
}

// UpdateLink edits an existing link without changing its identity, label or kind.
func (uc *TrackingUsecase) UpdateLink(ctx context.Context, userID, showID, linkID int64, value string) error {
	us, err := uc.repo.GetUserShow(ctx, userID, showID)
	if err != nil {
		return notTracked(err)
	}
	return notTracked(uc.repo.UpdateLink(ctx, us.ID, linkID, value))
}
