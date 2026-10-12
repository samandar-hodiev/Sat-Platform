package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	ok := map[string]string{
		"a@example.com":           "a@example.com",
		"  Learner@Example.COM  ": "learner@example.com",
		"first.last+sat@mail.uz":  "first.last+sat@mail.uz",
	}
	for in, want := range ok {
		got, err := NormalizeEmail(in)
		if err != nil || got != want {
			t.Errorf("NormalizeEmail(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	bad := []string{
		"", "   ", "plain", "@example.com", "a@", "a@localhost", "a b@example.com",
		"Name <a@example.com>", "a@example.com, b@example.com",
		strings.Repeat("a", 250) + "@x.uz",
	}
	for _, in := range bad {
		if _, err := NormalizeEmail(in); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("NormalizeEmail(%q) = %v; want ErrInvalidEmail", in, err)
		}
	}
}
