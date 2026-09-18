package usecase

import (
	"context"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/pkg/provider"
)

// RatingsRepo is the storage dependency of RatingsUsecase.
type RatingsRepo interface {
	UpsertRatings(ctx context.Context, ratings []entity.ShowRating) error
}

// RatingsUsecase fetches and stores aggregated show ratings.
type RatingsUsecase struct {
	provider provider.RatingProvider
	repo     RatingsRepo
}

// NewRatingsUsecase creates a RatingsUsecase.
func NewRatingsUsecase(p provider.RatingProvider, repo RatingsRepo) *RatingsUsecase {
	return &RatingsUsecase{provider: p, repo: repo}
}

// RefreshForShow fetches ratings for the show (by title/year) and upserts them.
func (uc *RatingsUsecase) RefreshForShow(ctx context.Context, show *entity.Show) error {
	year := 0
	if show.FirstAirDate != nil {
		year = show.FirstAirDate.Year()
	}
	prs, err := uc.provider.Ratings(ctx, show.Title, year)
	if err != nil {
		return err
	}
	if len(prs) == 0 {
		return nil
	}
	now := time.Now()
	ratings := make([]entity.ShowRating, 0, len(prs))
	for _, r := range prs {
		ratings = append(ratings, entity.ShowRating{
			ShowID:    show.ID,
			Source:    r.Source,
			Value:     r.Value,
			Votes:     r.Votes,
			FetchedAt: now,
		})
	}
	return uc.repo.UpsertRatings(ctx, ratings)
}
