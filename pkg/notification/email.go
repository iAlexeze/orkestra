// pkg/notification/email.go
//
// Email notification dispatch via SMTP.
// SMTP credentials are read from pkg/konfig env vars:
//
//	SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM
//
// Recipients are declared per team in the Katalog notification config.
// Multiple recipients receive one email each (BCC is not used — per-recipient
// tracking allows future per-recipient suppression).
package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/logger"
)

const emailDialTimeout = 10 * time.Second

// SMTPConfig holds the SMTP connection parameters read from pkg/konfig.
// Constructed by the caller (notification.Notifier) from konfig at startup.
type SMTPConfig struct {
	Host string
	Port string // default "587"
	User string
	Pass string
	From string
}

// IsConfigured returns true when the minimum required SMTP fields are present.
func (c SMTPConfig) IsConfigured() bool {
	return c.Host != "" && c.User != "" && c.Pass != "" && c.From != ""
}

// EffectivePort returns the port, defaulting to 587 (STARTTLS submission).
func (c SMTPConfig) EffectivePort() string {
	if c.Port == "" {
		return "587"
	}
	return c.Port
}

// sendEmailNotification sends a plain-text email to all recipients in the list.
// Each recipient receives an individual SMTP DATA transaction — this allows
// per-recipient delivery tracking and avoids exposing recipient lists.
func sendEmailNotification(
	ctx context.Context,
	cfg SMTPConfig,
	recipients []string,
	katalogName string,
	teamName string,
	subject string,
	body string,
) error {
	if !cfg.IsConfigured() {
		return fmt.Errorf("email: SMTP not configured (check SMTP_HOST, SMTP_USER, SMTP_PASS, SMTP_FROM)")
	}
	if len(recipients) == 0 {
		return nil
	}

	addr := net.JoinHostPort(cfg.Host, cfg.EffectivePort())

	// Dial with timeout — SMTP servers can be slow
	dialer := &net.Dialer{Timeout: emailDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("email: connecting to %s: %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("email: creating SMTP client: %w", err)
	}
	defer client.Close()

	// Upgrade to TLS via STARTTLS
	tlsCfg := &tls.Config{ServerName: cfg.Host}
	if err := client.StartTLS(tlsCfg); err != nil {
		// Log but continue — some internal SMTP servers don't support STARTTLS
		logger.Warn().Err(err).Str("host", cfg.Host).Msg("notification: STARTTLS failed, continuing without TLS")
	}

	// Authenticate
	auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("email: authentication failed: %w", err)
	}

	// Send to each recipient
	for _, to := range recipients {
		to = strings.TrimSpace(to)
		if to == "" {
			continue
		}
		if err := sendOne(client, cfg.From, to, katalogName, teamName, subject, body); err != nil {
			logger.Warn().Err(err).Str("to", to).Msg("notification: email delivery failed")
		} else {
			logger.Debug().Str("to", to).Str("team", teamName).Msg("notification: email sent")
		}
	}

	return client.Quit()
}

// sendOne sends a single email message within an existing SMTP session.
func sendOne(client *smtp.Client, from, to, katalogName, teamName, subject, body string) error {
	if err := client.Reset(); err != nil {
		return fmt.Errorf("reset: %w", err)
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	defer wc.Close()

	msg := buildEmailMessage(from, to, subject, body, katalogName, teamName)
	_, err = fmt.Fprint(wc, msg)
	return err
}

// buildEmailMessage constructs the RFC 5322 email message.
func buildEmailMessage(from, to, subject, body, katalogName, teamName string) string {
	return strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: [orkestra] " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"X-Orkestra-Katalog: " + katalogName,
		"X-Orkestra-Team: " + teamName,
		"",
		body,
	}, "\r\n")
}
