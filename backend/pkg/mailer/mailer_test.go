package mailer

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/deface90/defshows/backend/pkg/config"
)

func TestBuildMessage_PlainText(t *testing.T) {
	msg := string(buildMessage("from@x.io", "to@x.io", "Тема", "just text", ""))
	if !strings.Contains(msg, "Content-Type: text/plain; charset=\"UTF-8\"") {
		t.Fatalf("plain message should be text/plain, got:\n%s", msg)
	}
	if strings.Contains(msg, "multipart/alternative") {
		t.Fatal("plain message must not be multipart")
	}
	if !strings.Contains(msg, "just text") {
		t.Fatal("body missing")
	}
	// Subject with Cyrillic must be RFC 2047 encoded (not raw).
	if strings.Contains(msg, "Subject: Тема") {
		t.Fatalf("subject should be encoded, got:\n%s", msg)
	}
}

func TestBuildMessage_Multipart(t *testing.T) {
	msg := string(buildMessage("from@x.io", "to@x.io", "Subj", "text body", "<p>html body</p>"))
	for _, want := range []string{
		"multipart/alternative; boundary=",
		"Content-Type: text/plain; charset=\"UTF-8\"",
		"Content-Type: text/html; charset=\"UTF-8\"",
		"text body",
		"<p>html body</p>",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("multipart message missing %q, got:\n%s", want, msg)
		}
	}
}

func TestNew_FallsBackToLogMailer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if _, ok := New(config.SMTP{}, logger).(*LogMailer); !ok {
		t.Fatal("unconfigured SMTP should yield a LogMailer")
	}
	if _, ok := New(config.SMTP{Host: "smtp.x.io", From: "from@x.io"}, logger).(*SMTPMailer); !ok {
		t.Fatal("configured SMTP should yield an SMTPMailer")
	}
}
