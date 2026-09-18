package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/pkg/provider"
)

type fakeRatingProvider struct {
	ratings []provider.Rating
}

func (f fakeRatingProvider) Ratings(context.Context, string, int) ([]provider.Rating, error) {
	return f.ratings, nil
}

type fakeRatingsRepo struct {
	upserted []entity.ShowRating
}

func (f *fakeRatingsRepo) UpsertRatings(_ context.Context, ratings []entity.ShowRating) error {
	f.upserted = append(f.upserted, ratings...)
	return nil
}

func TestRatingsUsecase_RefreshForShow(t *testing.T) {
	votes := int64(100)
	fp := fakeRatingProvider{ratings: []provider.Rating{
		{Source: "imdb", Value: "9.2/10", Votes: &votes},
		{Source: "metacritic", Value: "86/100"},
	}}
	repo := &fakeRatingsRepo{}
	uc := usecase.NewRatingsUsecase(fp, repo)

	air := time.Date(2011, 4, 17, 0, 0, 0, 0, time.UTC)
	show := &entity.Show{ID: 7, Title: "Game of Thrones", FirstAirDate: &air}

	if err := uc.RefreshForShow(context.Background(), show); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if len(repo.upserted) != 2 {
		t.Fatalf("want 2 upserted, got %d", len(repo.upserted))
	}
	for _, r := range repo.upserted {
		if r.ShowID != 7 || r.FetchedAt.IsZero() {
			t.Fatalf("unexpected rating: %+v", r)
		}
	}
}

func TestRatingsUsecase_NoRatings(t *testing.T) {
	repo := &fakeRatingsRepo{}
	uc := usecase.NewRatingsUsecase(fakeRatingProvider{}, repo)
	if err := uc.RefreshForShow(context.Background(), &entity.Show{ID: 1, Title: "X"}); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if len(repo.upserted) != 0 {
		t.Fatalf("want no upserts, got %d", len(repo.upserted))
	}
}
