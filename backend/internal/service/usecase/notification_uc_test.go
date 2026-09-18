package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
)

type fakeNotifRepo struct {
	prefs   map[int64]*entity.NotificationPref
	tokens  map[string]int64
	read    map[int64]bool  // notification id -> read
	owner   map[int64]int64 // notification id -> user id
	allRead map[int64]bool  // user id -> MarkAllRead called
}

func newFakeNotifRepo() *fakeNotifRepo {
	return &fakeNotifRepo{
		prefs:   map[int64]*entity.NotificationPref{},
		tokens:  map[string]int64{},
		read:    map[int64]bool{},
		owner:   map[int64]int64{},
		allRead: map[int64]bool{},
	}
}

func (f *fakeNotifRepo) GetPrefs(_ context.Context, userID int64) (*entity.NotificationPref, error) {
	if p, ok := f.prefs[userID]; ok {
		return p, nil
	}
	def := entity.DefaultNotificationPref(userID)
	return &def, nil
}
func (f *fakeNotifRepo) UpsertPrefs(_ context.Context, p *entity.NotificationPref) error {
	f.prefs[p.UserID] = p
	return nil
}
func (f *fakeNotifRepo) ListForUser(context.Context, int64, int) ([]entity.Notification, error) {
	return nil, nil
}
func (f *fakeNotifRepo) MarkRead(_ context.Context, userID, id int64) error {
	if f.owner[id] != userID {
		return repository.ErrNotFound
	}
	f.read[id] = true
	return nil
}
func (f *fakeNotifRepo) MarkAllRead(_ context.Context, userID int64) error {
	f.allRead[userID] = true
	return nil
}
func (f *fakeNotifRepo) CreateLinkToken(_ context.Context, t *entity.TelegramLinkToken) error {
	f.tokens[t.Token] = t.UserID
	return nil
}
func (f *fakeNotifRepo) ConsumeLinkToken(_ context.Context, token string) (int64, error) {
	uid, ok := f.tokens[token]
	if !ok {
		return 0, repository.ErrNotFound
	}
	delete(f.tokens, token)
	return uid, nil
}

type fakeLinker struct {
	linked map[int64]int64 // userID -> chatID
}

func (f *fakeLinker) SetTelegramChatID(_ context.Context, userID, chatID int64) error {
	if f.linked == nil {
		f.linked = map[int64]int64{}
	}
	f.linked[userID] = chatID
	return nil
}

func (f *fakeLinker) TelegramChatID(_ context.Context, userID int64) (*int64, error) {
	if chatID, ok := f.linked[userID]; ok {
		return &chatID, nil
	}
	return nil, nil
}

func TestNotificationUsecase_TelegramLinkFlow(t *testing.T) {
	repo := newFakeNotifRepo()
	linker := &fakeLinker{}
	uc := usecase.NewNotificationUsecase(repo, linker, "defShowsBot", 0)
	ctx := context.Background()

	link, err := uc.GenerateTelegramLink(ctx, 7)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.HasPrefix(link, "https://t.me/defShowsBot?start=") {
		t.Fatalf("unexpected link: %s", link)
	}
	token := strings.TrimPrefix(link, "https://t.me/defShowsBot?start=")

	// Link with a chat id.
	if err := uc.LinkTelegram(ctx, token, 999); err != nil {
		t.Fatalf("link: %v", err)
	}
	if linker.linked[7] != 999 {
		t.Fatalf("expected user 7 linked to chat 999, got %v", linker.linked)
	}

	// Token is one-time → reuse fails.
	if err := uc.LinkTelegram(ctx, token, 999); !errors.Is(err, usecase.ErrInvalidLinkToken) {
		t.Fatalf("want ErrInvalidLinkToken on reuse, got %v", err)
	}
}

func TestNotificationUsecase_IsTelegramLinked(t *testing.T) {
	repo := newFakeNotifRepo()
	linker := &fakeLinker{}
	uc := usecase.NewNotificationUsecase(repo, linker, "bot", 0)
	ctx := context.Background()

	if linked, err := uc.IsTelegramLinked(ctx, 3); err != nil || linked {
		t.Fatalf("want unlinked, got linked=%v err=%v", linked, err)
	}
	_ = linker.SetTelegramChatID(ctx, 3, 42)
	if linked, err := uc.IsTelegramLinked(ctx, 3); err != nil || !linked {
		t.Fatalf("want linked, got linked=%v err=%v", linked, err)
	}
}

func TestNotificationUsecase_MarkRead(t *testing.T) {
	repo := newFakeNotifRepo()
	uc := usecase.NewNotificationUsecase(repo, &fakeLinker{}, "bot", 0)
	ctx := context.Background()

	repo.owner[10] = 1 // notification 10 belongs to user 1

	// Wrong user → not found.
	if err := uc.MarkRead(ctx, 2, 10); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound for other user, got %v", err)
	}
	// Owner → marked read.
	if err := uc.MarkRead(ctx, 1, 10); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if !repo.read[10] {
		t.Fatalf("notification 10 not marked read")
	}
	// Mark-all delegates to repo.
	if err := uc.MarkAllRead(ctx, 1); err != nil || !repo.allRead[1] {
		t.Fatalf("mark all read failed: err=%v called=%v", err, repo.allRead[1])
	}
}

func TestNotificationUsecase_UpdatePrefs(t *testing.T) {
	repo := newFakeNotifRepo()
	uc := usecase.NewNotificationUsecase(repo, &fakeLinker{}, "bot", 0)
	ctx := context.Background()

	off := false
	lead := 48
	p, err := uc.UpdatePrefs(ctx, 5, &off, nil, nil, &lead)
	if err != nil {
		t.Fatalf("update prefs: %v", err)
	}
	if p.EpisodeRelease != false || p.LeadTimeHours != 48 {
		t.Fatalf("prefs not applied: %+v", p)
	}
	// Persisted.
	got, _ := uc.GetPrefs(ctx, 5)
	if got.EpisodeRelease != false || got.LeadTimeHours != 48 {
		t.Fatalf("prefs not persisted: %+v", got)
	}
}
