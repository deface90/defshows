package usecase_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/usecase"
)

type progressRepo struct {
	usecase.TrackingRepo
	watched []int64
}

func (r progressRepo) GetUserShow(context.Context, int64, int64) (*entity.UserShow, error) {
	return &entity.UserShow{ID: 1, ShowID: 10}, nil
}
func (r progressRepo) WatchedEpisodeIDs(context.Context, int64) ([]int64, error) {
	return r.watched, nil
}

type progressCatalog struct {
	usecase.Catalog
	episodes []entity.Episode
}

func (c progressCatalog) Episodes(context.Context, int64) ([]entity.Episode, error) {
	return c.episodes, nil
}

func TestProgressCountsOnlyAiredEpisodes(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	past, future := today.AddDate(0, 0, -1), today.AddDate(0, 0, 2)
	episodes := []entity.Episode{
		{ID: 1, AirDate: &past}, {ID: 2, AirDate: &today},
		{ID: 3, AirDate: &future}, {ID: 4},
	}
	for _, tc := range []struct {
		name                    string
		episodes                []entity.Episode
		watched                 []int64
		total, count, unwatched int
		next                    int64
	}{
		{name: "today counts, future and unknown do not", episodes: episodes, total: 2, unwatched: 2, next: 1},
		{name: "all released watched is 100 percent", episodes: episodes, watched: []int64{1, 2}, total: 2, count: 2},
		{name: "out of order marks", episodes: episodes, watched: []int64{2}, total: 2, count: 1, unwatched: 1, next: 1},
		{name: "legacy future unknown and orphan marks excluded", episodes: episodes, watched: []int64{1, 3, 4, 99}, total: 2, count: 1, unwatched: 1, next: 2},
		{name: "nothing aired", episodes: episodes[2:], watched: []int64{3}},
		{name: "empty catalog"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uc := usecase.NewTrackingUsecase(progressRepo{watched: tc.watched}, progressCatalog{episodes: tc.episodes})
			got, err := uc.GetProgress(t.Context(), 1, 10)
			if err != nil {
				t.Fatal(err)
			}
			if got.Total != tc.total || got.Watched != tc.count || got.Unwatched != tc.unwatched {
				t.Fatalf("unexpected counters: %+v", got)
			}
			var next int64
			if got.NextUnwatched != nil {
				next = got.NextUnwatched.ID
			}
			if next != tc.next {
				t.Fatalf("next=%d, want %d", next, tc.next)
			}
			if !reflect.DeepEqual(got.WatchedEpisodeIDs, append([]int64{}, tc.watched...)) {
				t.Fatalf("saved marks changed: %v", got.WatchedEpisodeIDs)
			}
		})
	}
}
