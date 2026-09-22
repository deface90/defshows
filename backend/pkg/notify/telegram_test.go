package notify_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deface90/defshows/backend/pkg/notify"
)

func TestTelegramChannel_Send(t *testing.T) {
	var gotChatID float64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/bottest-token/sendMessage") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		gotChatID, _ = payload["chat_id"].(float64)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ch := notify.NewTelegramChannel(srv.URL, "test-token", srv.Client())
	if err := ch.Send(context.Background(), notify.Message{Target: "42", Body: "hi"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if int64(gotChatID) != 42 {
		t.Fatalf("chat id: %v", gotChatID)
	}
}

func TestTelegramChannel_Send_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ch := notify.NewTelegramChannel(srv.URL, "t", srv.Client())
	if err := ch.Send(context.Background(), notify.Message{Target: "1", Body: "x"}); err == nil {
		t.Fatal("expected error on 500")
	}
}
