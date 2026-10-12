package profile

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeName(t *testing.T) {
	ok := map[string]string{
		"  Ali  ":                "Ali",
		"Anne   Marie":           "Anne Marie",
		"O'Neil":                 "O'Neil",
		"Jean-Luc":               "Jean-Luc",
		"Мухаммад":               "Мухаммад",
		"Sr.":                    "Sr.",
		"Oʻrinboy":               "", // ʻ (U+02BB) is a modifier letter: allowed
		"José":                   "José",
		"\tLola\n":               "Lola",
		strings.Repeat("a", 100): strings.Repeat("a", 100),
	}
	for in, want := range ok {
		if want == "" {
			want = in
		}
		got, msg := normalizeName(in)
		if msg != "" || got != want {
			t.Errorf("normalizeName(%q) = %q, %q; want %q", in, got, msg, want)
		}
	}
	for _, in := range []string{"", "   ", "Ali2", "-Ali", "<b>", "Ali 😀", strings.Repeat("a", 101)} {
		if _, msg := normalizeName(in); msg == "" {
			t.Errorf("normalizeName(%q) accepted", in)
		}
	}
}

func TestNormalizePhone(t *testing.T) {
	ok := map[string]string{
		"":                   "",
		"+998 90 123-45-67":  "+998901234567",
		"00998901234567":     "+998901234567",
		"+1 (415) 555-0100":  "+14155550100",
		" +44 20 7946 0958 ": "+442079460958",
	}
	for in, want := range ok {
		got, msg := normalizePhone(in)
		if msg != "" || got != want {
			t.Errorf("normalizePhone(%q) = %q, %q; want %q", in, got, msg, want)
		}
	}
	for _, in := range []string{"901234567", "+0123456789", "+998 90 abc", "+12", "+9989012345678901", "998+901234567"} {
		if _, msg := normalizePhone(in); msg == "" {
			t.Errorf("normalizePhone(%q) accepted", in)
		}
	}
}

func TestParseDateOfBirth(t *testing.T) {
	today := time.Date(2026, 10, 12, 15, 0, 0, 0, time.UTC)
	if d, msg := parseDateOfBirth("2013-10-12", today); msg != "" || d.Format(time.DateOnly) != "2013-10-12" {
		t.Fatalf("13th birthday today: %v %q", d, msg)
	}
	for in, want := range map[string]string{
		"":           "is required",
		"12.10.2008": "must be a date in the format YYYY-MM-DD",
		"2008-02-30": "must be a date in the format YYYY-MM-DD",
		"1899-12-31": "must be a real date of birth",
		"2026-10-13": "must be a real date of birth",
		"2013-10-13": "you must be at least 13 years old to use SAT Platform",
	} {
		if _, msg := parseDateOfBirth(in, today); msg != want {
			t.Errorf("parseDateOfBirth(%q) = %q, want %q", in, msg, want)
		}
	}
}

func TestAgeOnLeapDay(t *testing.T) {
	dob := time.Date(2012, 2, 29, 0, 0, 0, 0, time.UTC)
	if got := ageOn(dob, time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)); got != 12 {
		t.Fatalf("28 Feb 2025: %d", got)
	}
	if got := ageOn(dob, time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)); got != 13 {
		t.Fatalf("1 Mar 2025: %d", got)
	}
}

func TestNormalizeTimezone(t *testing.T) {
	if tz, msg := normalizeTimezone("Asia/Tashkent"); msg != "" || tz != "Asia/Tashkent" {
		t.Fatal(tz, msg)
	}
	if tz, msg := normalizeTimezone(""); msg != "" || tz != "" {
		t.Fatal(tz, msg)
	}
	for _, in := range []string{"Local", "Mars/Olympus", "../etc/passwd"} {
		if _, msg := normalizeTimezone(in); msg == "" {
			t.Errorf("normalizeTimezone(%q) accepted", in)
		}
	}
}
