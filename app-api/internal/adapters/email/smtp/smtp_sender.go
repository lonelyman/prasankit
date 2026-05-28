// Package smtp provides an email.Sender implementation backed by stdlib net/smtp.
//
// TLS note: net/smtp negotiates STARTTLS automatically when the server advertises the
// STARTTLS capability (RFC 3207). A dedicated TLS config flag (e.g. implicit TLS on
// port 465 via tls.Dial) is left for a future hardening pass — not needed for Mailpit
// dev or STARTTLS-capable production servers.
package smtp

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"prasankit-api/internal/modules/email"
)

// Sender sends email via SMTP using stdlib net/smtp.
// Auth is nil when Username is empty (e.g. Mailpit dev has no auth).
// When Username is non-empty, smtp.PlainAuth is used.
type Sender struct {
	host        string // SMTP hostname
	port        string // SMTP port (string for net.JoinHostPort)
	fromAddress string // envelope / From: address
	fromName    string // optional display name — produces "Name <addr>" when set
	username    string // SMTP auth username (empty = no auth)
	password    string // SMTP auth password
}

// Config holds all fields required to construct a Sender.
type Config struct {
	Host        string
	Port        string
	FromAddress string
	FromName    string
	Username    string
	Password    string
}

// New creates a Sender from Config.
func New(cfg Config) *Sender {
	return &Sender{
		host:        cfg.Host,
		port:        cfg.Port,
		fromAddress: cfg.FromAddress,
		fromName:    cfg.FromName,
		username:    cfg.Username,
		password:    cfg.Password,
	}
}

// Send implements email.Sender.
// It builds a minimal RFC 822 message and delivers it via smtp.SendMail.
func (s *Sender) Send(_ context.Context, msg email.Message) error {
	addr := net.JoinHostPort(s.host, s.port)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	raw, err := buildMessage(s.fromAddress, s.fromName, msg)
	if err != nil {
		return fmt.Errorf("smtp sender: build message: %w", err)
	}

	if err := smtp.SendMail(addr, auth, s.fromAddress, []string{msg.To}, raw); err != nil {
		return fmt.Errorf("smtp sender: send: %w", err)
	}
	return nil
}

// buildMessage constructs a minimal RFC 822 / MIME message.
// If HTMLBody is non-empty, a multipart/alternative body is produced with
// text/plain first and text/html second (clients prefer last matching part).
// Otherwise, a simple text/plain message is produced.
func buildMessage(fromAddr, fromName string, msg email.Message) ([]byte, error) {
	var buf bytes.Buffer

	from := fromAddr
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromAddr)
	}

	now := time.Now().UTC().Format(time.RFC1123Z)

	if msg.HTMLBody == "" {
		// Simple text/plain message.
		fmt.Fprintf(&buf, "From: %s\r\n", from)
		fmt.Fprintf(&buf, "To: %s\r\n", msg.To)
		fmt.Fprintf(&buf, "Subject: %s\r\n", msg.Subject)
		fmt.Fprintf(&buf, "Date: %s\r\n", now)
		fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
		fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
		fmt.Fprintf(&buf, "\r\n")
		fmt.Fprintf(&buf, "%s", msg.TextBody)
	} else {
		// Multipart/alternative: text/plain + text/html.
		boundary := "==prasankit_boundary_001=="
		fmt.Fprintf(&buf, "From: %s\r\n", from)
		fmt.Fprintf(&buf, "To: %s\r\n", msg.To)
		fmt.Fprintf(&buf, "Subject: %s\r\n", msg.Subject)
		fmt.Fprintf(&buf, "Date: %s\r\n", now)
		fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
		fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary)
		fmt.Fprintf(&buf, "\r\n")

		// text/plain part
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: text/plain; charset=UTF-8\r\n")
		fmt.Fprintf(&buf, "\r\n")
		fmt.Fprintf(&buf, "%s\r\n", msg.TextBody)

		// text/html part
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: text/html; charset=UTF-8\r\n")
		fmt.Fprintf(&buf, "\r\n")
		fmt.Fprintf(&buf, "%s\r\n", msg.HTMLBody)

		// closing boundary
		fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	}

	return buf.Bytes(), nil
}
