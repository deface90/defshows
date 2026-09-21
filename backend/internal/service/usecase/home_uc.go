package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
)

// homeCacheTTL is how long a computed taste/recommendations payload is served
// before it is recomputed on the next request.
const homeCacheTTL = 12 * time.Hour

// Home computation limits.
const (
	topGenresN = 5
	topLangsN  = 5
	recsN      = 12
)

// GenreTaste / LangTaste / Taste describe a user's weighted preferences.
type GenreTaste struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

type LangTaste struct {
	Code   string `json:"code"`
	Weight int    `json:"weight"`
}

type Taste struct {
	Genres    []GenreTaste `json:"genres"`
	Languages []LangTaste  `json:"languages"`
}

// Home is the cacheable part of the personal home: taste profile plus catalog
// recommendations. Progress blocks are assembled live elsewhere.
type Home struct {
	Taste           Taste         `json:"taste"`
	Recommendations []entity.Show `json:"recommendations"`
}

// HomeRepo is the storage dependency of HomeUsecase.
type HomeRepo interface {
	TopGenres(ctx context.Context, userID int64, limit int) ([]repository.GenreWeight, error)
	TopLanguages(ctx context.Context, userID int64, limit int) ([]repository.LangWeight, error)
	Recommendations(ctx context.Context, userID int64, genreIDs []int64, limit int) ([]entity.Show, error)
	GetHomeCache(ctx context.Context, userID int64) (*entity.UserHomeCache, error)
	UpsertHomeCache(ctx context.Context, c *entity.UserHomeCache) error
}

// HomeUsecase builds a user's taste profile and recommendations from the local
// catalog, caching the result per user with a TTL.
type HomeUsecase struct {
	repo HomeRepo
	now  func() time.Time
}

// NewHomeUsecase creates a HomeUsecase.
func NewHomeUsecase(repo HomeRepo) *HomeUsecase {
	return &HomeUsecase{repo: repo, now: time.Now}
}

// TasteAndRecommendations returns the cached home if still fresh, otherwise
// recomputes it from the catalog and refreshes the cache.
func (uc *HomeUsecase) TasteAndRecommendations(ctx context.Context, userID int64) (*Home, error) {
	if cached, ok := uc.fresh(ctx, userID); ok {
		return cached, nil
	}
	home, err := uc.compute(ctx, userID)
	if err != nil {
		return nil, err
	}
	uc.store(ctx, userID, home)
	return home, nil
}

// fresh returns the cached home when present and within the TTL.
func (uc *HomeUsecase) fresh(ctx context.Context, userID int64) (*Home, bool) {
	c, err := uc.repo.GetHomeCache(ctx, userID)
	if err != nil || uc.now().Sub(c.ComputedAt) >= homeCacheTTL {
		return nil, false
	}
	var home Home
	if err := json.Unmarshal(c.Payload, &home); err != nil {
		return nil, false
	}
	return &home, true
}

// compute rebuilds the taste profile and recommendations from the catalog.
func (uc *HomeUsecase) compute(ctx context.Context, userID int64) (*Home, error) {
	genres, err := uc.repo.TopGenres(ctx, userID, topGenresN)
	if err != nil {
		return nil, err
	}
	langs, err := uc.repo.TopLanguages(ctx, userID, topLangsN)
	if err != nil {
		return nil, err
	}
	genreIDs := make([]int64, len(genres))
	taste := Taste{Genres: make([]GenreTaste, len(genres)), Languages: make([]LangTaste, len(langs))}
	for i, g := range genres {
		genreIDs[i] = g.ID
		taste.Genres[i] = GenreTaste{ID: g.ID, Name: g.Name, Weight: g.Weight}
	}
	for i, l := range langs {
		taste.Languages[i] = LangTaste{Code: l.Code, Weight: l.Weight}
	}
	recs, err := uc.repo.Recommendations(ctx, userID, genreIDs, recsN)
	if err != nil {
		return nil, err
	}
	if recs == nil {
		recs = []entity.Show{}
	}
	return &Home{Taste: taste, Recommendations: recs}, nil
}

// store persists the computed home; cache-write failures are non-fatal.
func (uc *HomeUsecase) store(ctx context.Context, userID int64, home *Home) {
	payload, err := json.Marshal(home)
	if err != nil {
		return
	}
	_ = uc.repo.UpsertHomeCache(ctx, &entity.UserHomeCache{
		UserID:     userID,
		Payload:    payload,
		ComputedAt: uc.now(),
	})
}
