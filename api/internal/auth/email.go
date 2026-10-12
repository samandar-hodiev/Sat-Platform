package auth

import (
	"errors"
	"net/mail"
	"strings"
)

// MaxEmailLength is the RFC 5321 limit for a forward path.
const MaxEmailLength = 254

var ErrInvalidEmail = errors.New("email address is not valid")

// NormalizeEmail trims and lower-cases an address and checks that it is a single plain
// address (no display name). Accounts are unique by the normalized form, so "A@x.com" and
// "a@x.com" are the same learner. Provider-specific rules (Gmail dots, "+tags") are not
// applied: they would merge mailboxes that really are different on other providers.
func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > MaxEmailLength {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || addr.Name != "" {
		return "", ErrInvalidEmail
	}
	at := strings.LastIndexByte(email, '@')
	if at < 1 || !strings.Contains(email[at+1:], ".") {
		return "", ErrInvalidEmail
	}
	return email, nil
}
