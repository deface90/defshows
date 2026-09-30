package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/usecase"
)

func ptrInt(v int) *int { return &v }

// TestNewFullSeason exercises the "caught up → whole new season dropped" signal.
func TestNewFullSeason(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	past, future := today.AddDate(0, 0, -1), today.AddDate(0, 0, 2)
	// S1: e1,e2 aired; S2: e3,e4 aired; plus an airing S3 (e5 aired, e6 future).
	full := []entity.Episode{
		{ID: 1, SeasonNumber: 1, EpisodeNumber: 1, AirDate: &past},
		{ID: 2, SeasonNumber: 1, EpisodeNumber: 2, AirDate: &past},
		{ID: 3, SeasonNumber: 2, EpisodeNumber: 1, AirDate: &past},
		{ID: 4, SeasonNumber: 2, EpisodeNumber: 2, AirDate: &past},
	}
	airingS2 := []entity.Episode{
		{ID: 1, SeasonNumber: 1, EpisodeNumber: 1, AirDate: &past},
		{ID: 2, SeasonNumber: 1, EpisodeNumber: 2, AirDate: &past},
		{ID: 3, SeasonNumber: 2, EpisodeNumber: 1, AirDate: &past},
		{ID: 4, SeasonNumber: 2, EpisodeNumber: 2, AirDate: &future},
	}
	for _, tc := range []struct {
		name     string
		episodes []entity.Episode
		watched  []int64
		want     *int
	}{
		{name: "caught up on S1, whole S2 dropped", episodes: full, watched: []int64{1, 2}, want: ptrInt(2)},
		{name: "fresh add, S1 fully aired untouched", episodes: full, watched: nil, want: ptrInt(1)},
		{name: "behind mid-S1, no badge", episodes: full, watched: []int64{1}, want: nil},
		{name: "all watched, no badge", episodes: full, watched: []int64{1, 2, 3, 4}, want: nil},
		{name: "next season still airing, not full", episodes: airingS2, watched: []int64{1, 2}, want: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uc := usecase.NewTrackingUsecase(progressRepo{watched: tc.watched}, progressCatalog{episodes: tc.episodes})
			got, err := uc.GetProgress(t.Context(), 1, 10)
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case tc.want == nil && got.NewFullSeason != nil:
				t.Fatalf("want nil, got season %d", *got.NewFullSeason)
			case tc.want != nil && got.NewFullSeason == nil:
				t.Fatalf("want season %d, got nil", *tc.want)
			case tc.want != nil && *got.NewFullSeason != *tc.want:
				t.Fatalf("want season %d, got %d", *tc.want, *got.NewFullSeason)
			}
		})
	}
}

type statsRepo struct {
	usecase.TrackingRepo
	shows   []entity.UserShow
	watched map[int64][]int64 // user_show_id -> watched episode ids
}

func (r statsRepo) ListUserShows(context.Context, int64, string) ([]entity.UserShow, error) {
	return r.shows, nil
}
func (r statsRepo) WatchedEpisodeIDs(_ context.Context, userShowID int64) ([]int64, error) {
	return r.watched[userShowID], nil
}

type statsCatalog struct {
	usecase.Catalog
	episodes map[int64][]entity.Episode // show_id -> episodes
}

func (c statsCatalog) Episodes(_ context.Context, showID int64) ([]entity.Episode, error) {
	return c.episodes[showID], nil
}

func TestStats(t *testing.T) {
	past := time.Now().UTC().AddDate(0, 0, -1)
	r30, r45 := 30, 45
	repo := statsRepo{
		shows: []entity.UserShow{
			{ID: 1, ShowID: 10, Status: entity.StatusCompleted},
			{ID: 2, ShowID: 20, Status: entity.StatusWatching},
		},
		watched: map[int64][]int64{
			1: {101, 102},
			2: {201},
		},
	}
	cat := statsCatalog{episodes: map[int64][]entity.Episode{
		10: {
			{ID: 101, SeasonNumber: 1, EpisodeNumber: 1, AirDate: &past, Runtime: &r30},
			{ID: 102, SeasonNumber: 1, EpisodeNumber: 2, AirDate: &past, Runtime: &r30},
		},
		20: {
			{ID: 201, SeasonNumber: 1, EpisodeNumber: 1, AirDate: &past, Runtime: &r45},
			{ID: 202, SeasonNumber: 1, EpisodeNumber: 2, AirDate: &past, Runtime: nil},
		},
	}}
	uc := usecase.NewTrackingUsecase(repo, cat)
	got, err := uc.Stats(t.Context(), 1)
	if err != nil {
		t.Fatal(err)
	}
	want := usecase.Stats{ShowsTracked: 2, ShowsCompleted: 1, SeasonsWatched: 1, EpisodesWatched: 3, MinutesWatched: 105}
	if got != want {
		t.Fatalf("stats: got %+v want %+v", got, want)
	}
}
