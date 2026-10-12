package mailer

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestNewLogRefusesProduction(t *testing.T) {
	if _, err := NewLog(slog.Default(), true); !errors.Is(err, ErrLogDriverInProduction) {
		t.Fatalf("want ErrLogDriverInProduction, got %v", err)
	}
}

func TestLogSendWritesMessage(t *testing.T) {
	var buf bytes.Buffer
	m, err := NewLog(slog.New(slog.NewTextHandler(&buf, nil)), false)
	if err != nil {
		t.Fatal(err)
	}
	err = m.Send(context.Background(), Message{To: "a@example.com", Subject: "Your code", Text: "Code: 123456"})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"a@example.com", "Your code", "123456"} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q: %s", want, out)
		}
	}
}

func TestLogSendValidates(t *testing.T) {
	m, _ := NewLog(slog.Default(), false)
	if err := m.Send(context.Background(), Message{To: "a@example.com"}); err == nil {
		t.Fatal("want error for missing subject and text")
	}
}
