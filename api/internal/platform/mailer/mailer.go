// Package mailer sends transactional email (verification codes, password reset links and,
// later, notifications from 3.14).
//
// Modules depend on the Mailer interface. Only the development "log" driver exists so far: it
// writes the message to the API log so a developer can read the code without an email
// provider. It refuses to run in production, so codes can never end up in production logs.
// A real provider (SMTP, Resend, SES) is added as another implementation chosen by
// configuration.
package mailer

import (
	"context"
	"errors"
	"log/slog"
)

// Message is one email. Text is required; HTML is optional and sent as the alternative part.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

var ErrLogDriverInProduction = errors.New("the log mail driver cannot be used in production")

// Log writes messages to the logger instead of sending them. Development and test only.
type Log struct{ log *slog.Logger }

func NewLog(log *slog.Logger, production bool) (*Log, error) {
	if production {
		return nil, ErrLogDriverInProduction
	}
	return &Log{log: log}, nil
}

func (l *Log) Send(ctx context.Context, msg Message) error {
	if msg.To == "" || msg.Subject == "" || msg.Text == "" {
		return errors.New("mailer: to, subject and text are required")
	}
	// The body is logged on purpose: it is the only way to read a code in development.
	// This driver cannot be constructed in production (see NewLog).
	l.log.InfoContext(ctx, "email (dev log driver, not sent)",
		slog.String("to", msg.To),
		slog.String("subject", msg.Subject),
		slog.String("text", msg.Text),
	)
	return nil
}
