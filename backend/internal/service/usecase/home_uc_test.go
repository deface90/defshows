package usecase_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
)

type fakeHomeRepo struct {
	cache    *entity.UserHomeCache
	cacheErr error

	genres []repository.GenreWeight
	langs  []repository.LangWeight
	recs   []entity.Show

	topGenresCalls int
	recGenreIDs    []int64
	upserted       *entity.UserHomeCache
}

func (f *fakeHomeRepo) TopGenres(_ context.Context, _ int64, _ int) ([]repository.GenreWeight, error) {
	f.topGenresCalls++
	return f.genres, nil
}
func (f *fakeHomeRepo) TopLanguages(_ context.Context, _ int64, _ int) ([]repository.LangWeight, error) {
	return f.langs, nil
}
func (f *fakeHomeRepo) Recommendations(_ context.Context, _ int64, genreIDs []int64, _ int) ([]entity.Show, error) {
	f.recGenreIDs = genreIDs
	return f.recs, nil
}
func (f *fakeHomeRepo) GetHomeCache(_ context.Context, _ int64) (*entity.UserHomeCache, error) {
	return f.cache, f.cacheErr
}
func (f *fakeHomeRepo) UpsertHomeCache(_ context.Context, c *entity.UserHomeCache) error {
	f.upserted = c
	return nil
}

func computeRepo() *fakeHomeRepo {
	return &fakeHomeRepo{
		cacheErr: repository.ErrNotFound,
		genres:   []repository.GenreWeight{{ID: 1, Name: "Драма", Weight: 5}, {ID: 2, Name: "Комедия", Weight: 3}},
		langs:    []repository.LangWeight{{Code: "ko", Weight: 4}},
		recs:     []entity.Show{{ID: 10, Title: "Recommended"}},
	}
}

func TestHomeRecomputesOnCacheMiss(t *testing.T) {
	repo := computeRepo()
	home, err := usecase.NewHomeUsecase(repo).TasteAndRecommendations(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if repo.topGenresCalls != 1 {
		t.Fatalf("want compute, TopGenres called %d times", repo.topGenresCalls)
	}
	if len(home.Taste.Genres) != 2 || home.Taste.Genres[0].Name != "Драма" {
		t.Fatalf("unexpected taste: %+v", home.Taste)
	}
	// Recommendations must be asked for exactly the top genre ids.
	if len(repo.recGenreIDs) != 2 || repo.recGenreIDs[0] != 1 || repo.recGenreIDs[1] != 2 {
		t.Fatalf("recommendations not scoped to top genres: %v", repo.recGenreIDs)
	}
	if len(home.Recommendations) != 1 || home.Recommendations[0].ID != 10 {
		t.Fatalf("unexpected recommendations: %+v", home.Recommendations)
	}
	if repo.upserted == nil {
		t.Fatal("computed home was not cached")
	}
}

func TestHomeServesFreshCacheWithoutRecomputing(t *testing.T) {
	payload, _ := json.Marshal(usecase.Home{
		Taste:           usecase.Taste{Genres: []usecase.GenreTaste{{ID: 9, Name: "Cached", Weight: 1}}},
		Recommendations: []entity.Show{},
	})
	repo := computeRepo()
	repo.cacheErr = nil
	repo.cache = &entity.UserHomeCache{UserID: 7, Payload: payload, ComputedAt: time.Now().Add(-1 * time.Hour)}

	home, err := usecase.NewHomeUsecase(repo).TasteAndRecommendations(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if repo.topGenresCalls != 0 {
		t.Fatal("fresh cache should not recompute")
	}
	if len(home.Taste.Genres) != 1 || home.Taste.Genres[0].Name != "Cached" {
		t.Fatalf("did not serve cached payload: %+v", home.Taste)
	}
}

func TestHomeRecomputesWhenCacheStale(t *testing.T) {
	repo := computeRepo()
	repo.cacheErr = nil
	repo.cache = &entity.UserHomeCache{UserID: 7, Payload: []byte(`{}`), ComputedAt: time.Now().Add(-13 * time.Hour)}

	if _, err := usecase.NewHomeUsecase(repo).TasteAndRecommendations(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if repo.topGenresCalls != 1 {
		t.Fatal("stale cache should trigger recompute")
	}
	if repo.upserted == nil {
		t.Fatal("recomputed home was not re-cached")
	}
}
