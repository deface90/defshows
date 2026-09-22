// Package mailer sends transactional email. It provides an SMTP implementation
// and a log-only fallback used when SMTP is not configured, so flows that send
// mail (e.g. password reset) also work locally without a mail server.
package mailer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"mime"
	"net/smtp"
	"strings"

	"github.com/deface90/defshows/backend/pkg/config"
)

// Mailer sends a transactional email to a single recipient. text is a plain-text
// fallback; html is the rich body (may be empty to send text only).
type Mailer interface {
	Send(ctx context.Context, to, subject, text, html string) error
}

// New returns an SMTPMailer when SMTP is configured, otherwise a LogMailer that
// logs messages instead of sending them.
func New(cfg config.SMTP, logger *slog.Logger) Mailer {
	if cfg.Enabled() {
		return &SMTPMailer{cfg: cfg}
	}
	logger.Warn("mailer: SMTP not configured, emails will be logged instead of sent")
	return &LogMailer{logger: logger}
}

// SMTPMailer sends mail through an SMTP server using the standard library
// (STARTTLS on the configured port — implicit-TLS ports like 465 are not
// supported by net/smtp's SendMail).
type SMTPMailer struct {
	cfg config.SMTP
}

// Send delivers a multipart/alternative (text + HTML) message. Auth is used only
// when a username is set. When html is empty a plain-text message is sent.
func (m *SMTPMailer) Send(_ context.Context, to, subject, text, html string) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	msg := buildMessage(m.cfg.From, to, subject, text, html)
	return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, msg)
}

// LogMailer logs messages rather than sending them (used when SMTP is unset).
type LogMailer struct {
	logger *slog.Logger
}

// Send logs the message so the link is visible in local development.
func (m *LogMailer) Send(_ context.Context, to, subject, text, _ string) error {
	m.logger.Info("mailer: email not sent (SMTP disabled)", "to", to, "subject", subject, "body", text)
	return nil
}

// buildMessage assembles an RFC 5322 message. With an HTML part it is
// multipart/alternative (text first, HTML second) so clients pick the richest
// renderable part; otherwise it is a single text/plain body. UTF-8 throughout.
func buildMessage(from, to, subject, text, html string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")

	if html == "" {
		b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
		b.WriteString(text)
		return []byte(b.String())
	}

	boundary := randomBoundary()
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
	b.WriteString(text + "\r\n")
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
	b.WriteString(html + "\r\n")
	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String())
}

func randomBoundary() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return "defshows-" + hex.EncodeToString(buf)
}
