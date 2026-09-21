package repository_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/testutil"
)

func TestHomeRepository(t *testing.T) {
	gdb := testutil.MigratedPostgresDB(t)
	testutil.Truncate(t, gdb, "users", "shows", "genres")
	ctx := context.Background()

	user := &entity.User{Email: strptr("home@example.com"), Role: entity.RoleUser, Timezone: "UTC"}
	if err := gdb.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	drama := &entity.Genre{TMDBID: 18, Name: "Драма"}
	comedy := &entity.Genre{TMDBID: 35, Name: "Комедия"}
	horror := &entity.Genre{TMDBID: 27, Name: "Хоррор"}
	for _, g := range []*entity.Genre{drama, comedy, horror} {
		if err := gdb.Create(g).Error; err != nil {
			t.Fatalf("seed genre: %v", err)
		}
	}

	// tracked: drama+comedy, Korean. recBoth: drama+comedy (2 overlap, low pop).
	// recDrama: drama only (1 overlap, high pop). recHorror: no overlap.
	tracked := &entity.Show{TMDBID: 1, Title: "Tracked", OriginalLanguage: "ko", AiringStatus: entity.AiringEnded, Genres: []entity.Genre{*drama, *comedy}}
	recBoth := &entity.Show{TMDBID: 2, Title: "RecBoth", Popularity: 1, AiringStatus: entity.AiringEnded, Genres: []entity.Genre{*drama, *comedy}}
	recDrama := &entity.Show{TMDBID: 3, Title: "RecDrama", Popularity: 99, AiringStatus: entity.AiringEnded, Genres: []entity.Genre{*drama}}
	recHorror := &entity.Show{TMDBID: 4, Title: "RecHorror", Popularity: 50, AiringStatus: entity.AiringEnded, Genres: []entity.Genre{*horror}}
	for _, s := range []*entity.Show{tracked, recBoth, recDrama, recHorror} {
		if err := gdb.Create(s).Error; err != nil {
			t.Fatalf("seed show: %v", err)
		}
	}
	if err := gdb.Create(&entity.UserShow{UserID: user.ID, ShowID: tracked.ID, Status: entity.StatusWatching}).Error; err != nil {
		t.Fatalf("seed user_show: %v", err)
	}

	repo := repository.NewHomeRepository(gdb)

	genres, err := repo.TopGenres(ctx, user.ID, 5)
	if err != nil {
		t.Fatalf("top genres: %v", err)
	}
	if len(genres) != 2 {
		t.Fatalf("want 2 taste genres, got %d: %+v", len(genres), genres)
	}
	// watching weight = 2 for each of the tracked show's genres.
	if genres[0].Weight != 2 {
		t.Fatalf("unexpected genre weight: %+v", genres[0])
	}

	langs, err := repo.TopLanguages(ctx, user.ID, 5)
	if err != nil {
		t.Fatalf("top languages: %v", err)
	}
	if len(langs) != 1 || langs[0].Code != "ko" || langs[0].Weight != 2 {
		t.Fatalf("unexpected languages: %+v", langs)
	}

	recs, err := repo.Recommendations(ctx, user.ID, []int64{drama.ID, comedy.ID}, 10)
	if err != nil {
		t.Fatalf("recommendations: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("want 2 recs (tracked + horror excluded), got %d: %+v", len(recs), recs)
	}
	// recBoth (2 genre overlaps) outranks recDrama (1 overlap) despite lower popularity.
	if recs[0].Title != "RecBoth" || recs[1].Title != "RecDrama" {
		t.Fatalf("unexpected recommendation order: %s, %s", recs[0].Title, recs[1].Title)
	}

	// Cache round-trip: missing, then upsert, then read, then update. jsonb
	// reformats the stored text, so compare the decoded value, not raw bytes.
	if _, err := repo.GetHomeCache(ctx, user.ID); err != repository.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if err := repo.UpsertHomeCache(ctx, &entity.UserHomeCache{UserID: user.ID, Payload: []byte(`{"v":1}`), ComputedAt: time.Now()}); err != nil {
		t.Fatalf("upsert cache: %v", err)
	}
	got, err := repo.GetHomeCache(ctx, user.ID)
	if err != nil || cacheVersion(t, got.Payload) != 1 {
		t.Fatalf("get cache: %v, %s", err, got.Payload)
	}
	if err := repo.UpsertHomeCache(ctx, &entity.UserHomeCache{UserID: user.ID, Payload: []byte(`{"v":2}`), ComputedAt: time.Now()}); err != nil {
		t.Fatalf("re-upsert cache: %v", err)
	}
	got, _ = repo.GetHomeCache(ctx, user.ID)
	if cacheVersion(t, got.Payload) != 2 {
		t.Fatalf("cache not updated: %s", got.Payload)
	}
}

func cacheVersion(t *testing.T, payload []byte) int {
	t.Helper()
	var v struct {
		V int `json:"v"`
	}
	if err := json.Unmarshal(payload, &v); err != nil {
		t.Fatalf("decode cache payload: %v", err)
	}
	return v.V
}
