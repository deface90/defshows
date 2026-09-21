package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/pkg/provider"
)

// ErrShowNotFound is returned when a show is not in the local catalog.
var ErrShowNotFound = errors.New("usecase: show not found")

// ImageMirror stores an external image under a deterministic key and returns
// the public URL to serve it from. Used to mirror TMDB posters/backdrops so
// browsers never hit the (blockable) TMDB image host.
type ImageMirror interface {
	Mirror(ctx context.Context, srcURL, key string) (string, error)
}

// CatalogRepo is the storage dependency of CatalogUsecase.
type CatalogRepo interface {
	UpsertShow(ctx context.Context, s *entity.Show) error
	GetShowByID(ctx context.Context, id int64) (*entity.Show, error)
	GetShowByTMDBID(ctx context.Context, tmdbID int64) (*entity.Show, error)
	GetShowWithRelations(ctx context.Context, id int64) (*entity.Show, error)
	UpsertSeasons(ctx context.Context, seasons []entity.Season) error
	GetSeasonsByShow(ctx context.Context, showID int64) ([]entity.Season, error)
	UpsertEpisodes(ctx context.Context, episodes []entity.Episode) error
	GetEpisodesByShow(ctx context.Context, showID int64) ([]entity.Episode, error)
	SetNextEpisode(ctx context.Context, showID int64, episodeID *int64, airDate *time.Time, status entity.AiringStatus) error
	UpsertGenres(ctx context.Context, genres []entity.Genre) error
	ReplaceShowGenres(ctx context.Context, showID int64, genreIDs []int64) error
	ListShows(ctx context.Context, query string, limit, offset int) ([]entity.Show, error)
	ShowsForRefresh(ctx context.Context, staleBefore time.Time, limit int) ([]entity.Show, error)
	GetRatingsByShow(ctx context.Context, showID int64) ([]entity.ShowRating, error)
}

// ShowDetail is a show plus its flat episode list and ratings.
type ShowDetail struct {
	Show     *entity.Show
	Episodes []entity.Episode
	Ratings  []entity.ShowRating
}

// CatalogUsecase imports and reads the mirrored catalog.
type CatalogUsecase struct {
	repo     CatalogRepo
	provider provider.ShowProvider
	mirror   ImageMirror
	logger   *slog.Logger
}

// NewCatalogUsecase creates a CatalogUsecase.
func NewCatalogUsecase(repo CatalogRepo, p provider.ShowProvider) *CatalogUsecase {
	return &CatalogUsecase{repo: repo, provider: p, logger: slog.Default()}
}

// WithImageMirror attaches an image mirror used to copy posters/backdrops into
// object storage on import. Returns the usecase for chaining.
func (uc *CatalogUsecase) WithImageMirror(m ImageMirror) *CatalogUsecase {
	uc.mirror = m
	return uc
}

// SearchExternal searches the external provider (TMDB).
func (uc *CatalogUsecase) SearchExternal(ctx context.Context, query string) ([]provider.ShowSummary, error) {
	return uc.provider.SearchShows(ctx, query)
}

// SearchLocal searches the local mirrored catalog.
func (uc *CatalogUsecase) SearchLocal(ctx context.Context, query string, limit, offset int) ([]entity.Show, error) {
	return uc.repo.ListShows(ctx, query, limit, offset)
}

// GetShow returns a locally-mirrored show with relations and episodes.
func (uc *CatalogUsecase) GetShow(ctx context.Context, id int64) (*ShowDetail, error) {
	show, err := uc.repo.GetShowWithRelations(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrShowNotFound
		}
		return nil, err
	}
	episodes, err := uc.repo.GetEpisodesByShow(ctx, id)
	if err != nil {
		return nil, err
	}
	ratings, err := uc.repo.GetRatingsByShow(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ShowDetail{Show: show, Episodes: episodes, Ratings: ratings}, nil
}

// ShowsForRefresh lists catalog shows that should be re-synced.
func (uc *CatalogUsecase) ShowsForRefresh(ctx context.Context, staleBefore time.Time, limit int) ([]entity.Show, error) {
	return uc.repo.ShowsForRefresh(ctx, staleBefore, limit)
}

// ShowByID returns a locally-mirrored show without relations.
func (uc *CatalogUsecase) ShowByID(ctx context.Context, id int64) (*entity.Show, error) {
	return uc.repo.GetShowByID(ctx, id)
}

// Episodes returns a show's episodes (ordered), for progress calculation.
func (uc *CatalogUsecase) Episodes(ctx context.Context, showID int64) ([]entity.Episode, error) {
	return uc.repo.GetEpisodesByShow(ctx, showID)
}

// EnsureShow returns the local show for a tmdbID, importing it on first use.
func (uc *CatalogUsecase) EnsureShow(ctx context.Context, tmdbID int64) (*entity.Show, error) {
	if s, err := uc.repo.GetShowByTMDBID(ctx, tmdbID); err == nil {
		return s, nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	return uc.ImportShow(ctx, tmdbID)
}

// ImportShow fetches a show (and its seasons/episodes) from the provider and
// mirrors it into the local catalog, recomputing airing status.
func (uc *CatalogUsecase) ImportShow(ctx context.Context, tmdbID int64) (*entity.Show, error) {
	ps, err := uc.provider.GetShow(ctx, tmdbID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	status := deriveAiringStatus(ps, now)
	posterURL := uc.mirrorImage(ctx, ps.PosterURL, fmt.Sprintf("shows/%d/poster.jpg", ps.TMDBID))
	backdropURL := uc.mirrorImage(ctx, ps.BackdropURL, fmt.Sprintf("shows/%d/backdrop.jpg", ps.TMDBID))
	show := &entity.Show{
		IMDbURL:            ps.IMDbURL,
		WikipediaURL:       ps.WikipediaURL,
		TMDBID:             ps.TMDBID,
		Title:              ps.Title,
		OriginalTitle:      ps.OriginalTitle,
		Overview:           ps.Overview,
		PosterKey:          strOrNil(posterURL),
		BackdropKey:        strOrNil(backdropURL),
		Status:             ps.Status,
		InProduction:       ps.InProduction,
		FirstAirDate:       ps.FirstAirDate,
		LastAirDate:        ps.LastAirDate,
		NextEpisodeAirDate: episodeAirDate(ps.NextEpisode),
		LastEpisodeAirDate: episodeAirDate(ps.LastEpisode),
		AiringStatus:       status,
		OriginalLanguage:   ps.OriginalLanguage,
		Popularity:         ps.Popularity,
		VoteAverage:        ps.VoteAverage,
		VoteCount:          ps.VoteCount,
		LastSyncedAt:       &now,
	}
	if err := uc.repo.UpsertShow(ctx, show); err != nil {
		return nil, err
	}

	if err := uc.importGenres(ctx, show.ID, ps.Genres); err != nil {
		return nil, err
	}
	if err := uc.importSeasonsAndEpisodes(ctx, show, ps); err != nil {
		return nil, err
	}
	if err := uc.linkNextEpisode(ctx, show.ID, ps, status); err != nil {
		return nil, err
	}
	return uc.repo.GetShowByID(ctx, show.ID)
}

func (uc *CatalogUsecase) importGenres(ctx context.Context, showID int64, pgenres []provider.Genre) error {
	if len(pgenres) == 0 {
		return nil
	}
	genres := make([]entity.Genre, 0, len(pgenres))
	for _, g := range pgenres {
		genres = append(genres, entity.Genre{TMDBID: g.TMDBID, Name: g.Name})
	}
	if err := uc.repo.UpsertGenres(ctx, genres); err != nil {
		return err
	}
	ids := make([]int64, 0, len(genres))
	for _, g := range genres {
		ids = append(ids, g.ID)
	}
	return uc.repo.ReplaceShowGenres(ctx, showID, ids)
}

func (uc *CatalogUsecase) importSeasonsAndEpisodes(ctx context.Context, show *entity.Show, ps *provider.Show) error {
	if len(ps.Seasons) == 0 {
		return nil
	}
	seasons := make([]entity.Season, 0, len(ps.Seasons))
	seasonDetails := make(map[int]*provider.Season, len(ps.Seasons))
	for _, s := range ps.Seasons {
		if s.SeasonNumber == 0 {
			continue // skip TMDB "Specials" (season 0)
		}
		sd, err := uc.provider.GetSeason(ctx, ps.TMDBID, s.SeasonNumber)
		if err != nil {
			return err
		}
		seasonDetails[s.SeasonNumber] = sd
		seasons = append(seasons, entity.Season{
			VoteAverage:  sd.VoteAverage,
			ShowID:       show.ID,
			TMDBID:       s.TMDBID,
			SeasonNumber: s.SeasonNumber,
			Name:         s.Name,
			Overview:     s.Overview,
			AirDate:      s.AirDate,
			EpisodeCount: s.EpisodeCount,
			PosterKey:    strOrNil(s.PosterURL),
		})
	}
	if err := uc.repo.UpsertSeasons(ctx, seasons); err != nil {
		return err
	}

	stored, err := uc.repo.GetSeasonsByShow(ctx, show.ID)
	if err != nil {
		return err
	}
	seasonID := make(map[int]int64, len(stored))
	for _, s := range stored {
		seasonID[s.SeasonNumber] = s.ID
	}

	var episodes []entity.Episode
	for _, s := range ps.Seasons {
		if s.SeasonNumber == 0 {
			continue // skip specials — no season row was created for them
		}
		sd := seasonDetails[s.SeasonNumber]
		sid, ok := seasonID[s.SeasonNumber]
		if !ok {
			continue
		}
		for _, e := range sd.Episodes {
			episodes = append(episodes, entity.Episode{
				VoteAverage:   e.VoteAverage,
				VoteCount:     e.VoteCount,
				SeasonID:      sid,
				ShowID:        show.ID,
				TMDBID:        e.TMDBID,
				SeasonNumber:  e.SeasonNumber,
				EpisodeNumber: e.EpisodeNumber,
				Name:          e.Name,
				Overview:      e.Overview,
				AirDate:       e.AirDate,
				Runtime:       e.Runtime,
				StillKey:      strOrNil(e.StillURL),
			})
		}
	}
	return uc.repo.UpsertEpisodes(ctx, episodes)
}

func (uc *CatalogUsecase) linkNextEpisode(ctx context.Context, showID int64, ps *provider.Show, status entity.AiringStatus) error {
	if ps.NextEpisode == nil {
		return uc.repo.SetNextEpisode(ctx, showID, nil, nil, status)
	}
	episodes, err := uc.repo.GetEpisodesByShow(ctx, showID)
	if err != nil {
		return err
	}
	for i := range episodes {
		e := episodes[i]
		if e.SeasonNumber == ps.NextEpisode.SeasonNumber && e.EpisodeNumber == ps.NextEpisode.EpisodeNumber {
			return uc.repo.SetNextEpisode(ctx, showID, &e.ID, e.AirDate, status)
		}
	}
	return uc.repo.SetNextEpisode(ctx, showID, nil, ps.NextEpisode.AirDate, status)
}

func deriveAiringStatus(ps *provider.Show, now time.Time) entity.AiringStatus {
	switch {
	case ps.NextEpisode != nil:
		if nextSeasonStarted(ps, now) {
			return entity.AiringNow
		}
		return entity.AiringBetweenSeasons
	case ps.InProduction:
		return entity.AiringBetweenSeasons
	case ps.Status == "Ended" || ps.Status == "Canceled":
		return entity.AiringEnded
	case ps.LastEpisode != nil:
		return entity.AiringBetweenSeasons
	default:
		return entity.AiringNotStarted
	}
}

// A scheduled episode does not mean its season has premiered. TMDB's season
// air date is the premiere date; never use the previous season's last episode
// as evidence that the upcoming season is already airing.
func nextSeasonStarted(ps *provider.Show, now time.Time) bool {
	next := ps.NextEpisode
	if next.SeasonNumber <= 0 {
		return false
	}
	for _, season := range ps.Seasons {
		if season.SeasonNumber == next.SeasonNumber && season.AirDate != nil {
			return aired(season.AirDate, now)
		}
	}
	// Fallback for incomplete season metadata, including premiere day before
	// TMDB moves the first episode from next_episode_to_air to last_episode_to_air.
	if next.EpisodeNumber == 1 && aired(next.AirDate, now) {
		return true
	}
	last := ps.LastEpisode
	return last != nil && last.SeasonNumber == next.SeasonNumber && last.EpisodeNumber > 0 && aired(last.AirDate, now)
}

func episodeAirDate(e *provider.Episode) *time.Time {
	if e == nil {
		return nil
	}
	return e.AirDate
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// mirrorImage copies srcURL into object storage under key and returns the public
// URL. With no mirror configured, or on any error, it falls back to srcURL so
// the catalog still records the (original) image reference. Deterministic keys
// mean re-imports overwrite in place — images don't accumulate.
func (uc *CatalogUsecase) mirrorImage(ctx context.Context, srcURL, key string) string {
	if uc.mirror == nil || srcURL == "" {
		return srcURL
	}
	publicURL, err := uc.mirror.Mirror(ctx, srcURL, key)
	if err != nil {
		uc.logger.Warn("image mirror failed", "key", key, "err", err)
		return srcURL
	}
	return publicURL
}
