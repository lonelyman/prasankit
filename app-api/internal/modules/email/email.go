// Package email defines the port (interface) for sending email in the Prasankit domain layer.
// It contains no SMTP or transport-specific imports — only pure domain types.
package email

import "context"

// Message is the domain representation of an outbound email.
// Consumers fill in the fields; the adapter handles encoding and transport.
type Message struct {
	To       string // recipient address
	Subject  string
	TextBody string // plain-text body
	HTMLBody string // optional HTML body; adapter may use multipart/alternative if set
}

// Sender is the port that application services depend on to send email.
// Adapters (e.g. smtp_sender) implement this interface.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
