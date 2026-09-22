// Package mailer sends transactional email. It provides an SMTP implementation
// and a log-only fallback used when SMTP is not configured, so flows that send
// mail (e.g. password reset) also work locally without a mail server.
package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/deface90/defshows/backend/pkg/config"
)

// Mailer sends a plain-text email to a single recipient.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
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

// SMTPMailer sends mail through an SMTP server using the standard library.
type SMTPMailer struct {
	cfg config.SMTP
}

// Send delivers a plain-text UTF-8 message. Auth is used only when a username is set.
func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	msg := buildMessage(m.cfg.From, to, subject, body)
	return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, msg)
}

// LogMailer logs messages rather than sending them (used when SMTP is unset).
type LogMailer struct {
	logger *slog.Logger
}

// Send logs the message so the link is visible in local development.
func (m *LogMailer) Send(_ context.Context, to, subject, body string) error {
	m.logger.Info("mailer: email not sent (SMTP disabled)", "to", to, "subject", subject, "body", body)
	return nil
}

// buildMessage assembles a minimal RFC 5322 message with UTF-8 headers.
func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}
