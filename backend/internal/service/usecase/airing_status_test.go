package usecase

import (
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/pkg/provider"
)

func TestDeriveAiringStatusSeasonPremiere(t *testing.T) {
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	today := time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)
	future, past := now.AddDate(0, 0, 7), now.AddDate(0, 0, -7)
	for _, tc := range []struct {
		name string
		show provider.Show
		want entity.AiringStatus
	}{
		{"announced new season after previous finale", provider.Show{
			NextEpisode: &provider.Episode{SeasonNumber: 2, EpisodeNumber: 1, AirDate: &future},
			LastEpisode: &provider.Episode{SeasonNumber: 1, EpisodeNumber: 10, AirDate: &past},
			Seasons:     []provider.Season{{SeasonNumber: 1, AirDate: &past}, {SeasonNumber: 2, AirDate: &future}},
		}, entity.AiringBetweenSeasons},
		{"premiere today", provider.Show{
			NextEpisode: &provider.Episode{SeasonNumber: 2, EpisodeNumber: 1, AirDate: &today},
			Seasons:     []provider.Season{{SeasonNumber: 2, AirDate: &today}},
		}, entity.AiringNow},
		{"season already started", provider.Show{
			NextEpisode: &provider.Episode{SeasonNumber: 2, EpisodeNumber: 3, AirDate: &future},
			Seasons:     []provider.Season{{SeasonNumber: 2, AirDate: &past}},
		}, entity.AiringNow},
		{"upcoming first season", provider.Show{
			NextEpisode: &provider.Episode{SeasonNumber: 1, EpisodeNumber: 1, AirDate: &future},
			Seasons:     []provider.Season{{SeasonNumber: 1, AirDate: &future}},
		}, entity.AiringBetweenSeasons},
		{"unknown premiere and only a previous-season episode", provider.Show{
			NextEpisode: &provider.Episode{SeasonNumber: 2, EpisodeNumber: 2, AirDate: &future},
			LastEpisode: &provider.Episode{SeasonNumber: 1, EpisodeNumber: 10, AirDate: &past},
		}, entity.AiringBetweenSeasons},
		{"unknown premiere but current-season episode released", provider.Show{
			NextEpisode: &provider.Episode{SeasonNumber: 2, EpisodeNumber: 2, AirDate: &future},
			LastEpisode: &provider.Episode{SeasonNumber: 2, EpisodeNumber: 1, AirDate: &past},
		}, entity.AiringNow},
		{"premiere fallback without season metadata", provider.Show{
			NextEpisode: &provider.Episode{SeasonNumber: 2, EpisodeNumber: 1, AirDate: &today},
		}, entity.AiringNow},
		{"unknown dates do not imply airing", provider.Show{
			NextEpisode: &provider.Episode{SeasonNumber: 2, EpisodeNumber: 2},
			LastEpisode: &provider.Episode{SeasonNumber: 2, EpisodeNumber: 1},
		}, entity.AiringBetweenSeasons},
		{"waiting without a scheduled episode", provider.Show{InProduction: true}, entity.AiringBetweenSeasons},
		{"ended", provider.Show{Status: "Ended"}, entity.AiringEnded},
		{"canceled", provider.Show{Status: "Canceled"}, entity.AiringEnded},
		{"no episodes yet", provider.Show{}, entity.AiringNotStarted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := deriveAiringStatus(&tc.show, now); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}
