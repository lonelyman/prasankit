package smtpemail

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"

	"prasankit-api/internal/config"
)

type Sender struct {
	host     string
	port     string
	username string
	password string
	from     mail.Address
	subject  string
}

func NewSender(cfg config.MailConfig) Sender {
	return Sender{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		from: mail.Address{
			Name:    cfg.FromName,
			Address: cfg.FromAddress,
		},
		subject: cfg.VerifyEmailSubject,
	}
}

func (s Sender) SendVerificationEmail(ctx context.Context, toEmail string, verificationURL string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	to := mail.Address{Address: toEmail}
	body := "Verify your Prasankit account\n\n" +
		"Open this link to verify your email address:\n" +
		verificationURL + "\n\n" +
		"If you did not create this account, ignore this email.\n"

	msg := bytes.Buffer{}
	writeHeader(&msg, "From", s.from.String())
	writeHeader(&msg, "To", to.String())
	writeHeader(&msg, "Subject", mime.QEncoding.Encode("UTF-8", s.subject))
	writeHeader(&msg, "MIME-Version", "1.0")
	writeHeader(&msg, "Content-Type", `text/plain; charset="UTF-8"`)
	writeHeader(&msg, "Content-Transfer-Encoding", "8bit")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	addr := net.JoinHostPort(s.host, s.port)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	if err := smtp.SendMail(addr, auth, s.from.Address, []string{to.Address}, msg.Bytes()); err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}
	return nil
}

func writeHeader(buf *bytes.Buffer, key string, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(strings.ReplaceAll(value, "\n", ""))
	buf.WriteString("\r\n")
}
