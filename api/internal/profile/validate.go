package profile

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	NameMaxLength = 100
	// MinimumAge is the youngest a learner may be (decided for 2.01: 13, the COPPA line).
	MinimumAge = 13
)

// Names: letters (any script) with spaces, hyphens, apostrophes and dots between them, e.g.
// "Anne-Marie", "O'Neil", "Jr.". Digits, emoji and control characters are rejected.
var namePattern = regexp.MustCompile(`^\p{L}[\p{L}\p{M} '’.\-]*$`)

// normalizeName trims and collapses inner whitespace. msg is set when the result is empty,
// too long or contains characters a name does not have.
func normalizeName(raw string) (name string, msg string) {
	name = strings.Join(strings.Fields(raw), " ")
	switch {
	case name == "":
		return "", "is required"
	case utf8.RuneCountInString(name) > NameMaxLength:
		return "", "must be at most 100 characters"
	case !namePattern.MatchString(name):
		return "", "can contain only letters, spaces, hyphens, apostrophes and dots"
	}
	return name, ""
}

var e164 = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

// normalizePhone accepts international numbers as people type them ("+998 90 123-45-67",
// "00998901234567") and returns E.164 (+998901234567). Empty input means no phone.
func normalizePhone(raw string) (phone string, msg string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ""
	}
	var b strings.Builder
	for i, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && i == 0:
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '(' || r == ')' || r == '.':
		default:
			return "", "must be an international number, e.g. +998 90 123 45 67"
		}
	}
	phone = b.String()
	if strings.HasPrefix(phone, "00") {
		phone = "+" + phone[2:]
	}
	if !e164.MatchString(phone) {
		return "", "must be an international number, e.g. +998 90 123 45 67"
	}
	return phone, ""
}

// parseDateOfBirth reads YYYY-MM-DD and checks the learner is at least MinimumAge on today's
// date (UTC). The date has no time zone: it is stored as a DATE.
func parseDateOfBirth(raw string, today time.Time) (time.Time, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, "is required"
	}
	dob, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return time.Time{}, "must be a date in the format YYYY-MM-DD"
	}
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if dob.Year() < 1900 || dob.After(today) {
		return time.Time{}, "must be a real date of birth"
	}
	if ageOn(dob, today) < MinimumAge {
		return time.Time{}, "you must be at least 13 years old to use SAT Platform"
	}
	return dob, ""
}

// ageOn returns full years between dob and day. A 29 February birthday counts on 1 March in
// non-leap years.
func ageOn(dob, day time.Time) int {
	age := day.Year() - dob.Year()
	if day.Month() < dob.Month() || (day.Month() == dob.Month() && day.Day() < dob.Day()) {
		age--
	}
	return age
}

// normalizeTimezone accepts an IANA name the server knows. Empty means "keep the current one".
func normalizeTimezone(raw string) (string, string) {
	tz := strings.TrimSpace(raw)
	if tz == "" {
		return "", ""
	}
	if len(tz) > 64 || tz == "Local" {
		return "", "must be an IANA time zone, e.g. Asia/Tashkent"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return "", "must be an IANA time zone, e.g. Asia/Tashkent"
	}
	return tz, ""
}
