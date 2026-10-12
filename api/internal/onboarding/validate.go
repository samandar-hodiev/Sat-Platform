package onboarding

import (
	"slices"
	"strings"
	"time"
)

// Answer ranges (decided for 2.02). The database CHECK constraints hold the same values.
const (
	ScoreMin, ScoreMax     = 400, 1600
	SectionMin, SectionMax = 200, 800
	// ScoreStep: SAT scores are multiples of 10.
	ScoreStep = 10
	// ExamDateMaxYears: an exam date further ahead than this is a typo.
	ExamDateMaxYears = 2
	ConfidenceMin    = 1
	ConfidenceMax    = 5
)

// Reasons for taking the SAT (optional, PRD §8.1).
var Reasons = []string{"college_admission", "scholarship", "improve_score", "practice", "other"}

// DailyMinutes are the daily study time choices; 120 reads "2 hours or more".
var DailyMinutes = []int{15, 30, 45, 60, 90, 120}

// firstScoreMonth: results before March 2016 were on the old 2400 scale.
var firstScoreMonth = time.Date(2016, time.March, 1, 0, 0, 0, 0, time.UTC)

func validScore(v, min, max int) bool { return v >= min && v <= max && v%ScoreStep == 0 }

// checkTargetScore: required, 400–1600 in steps of 10.
func checkTargetScore(v *int) string {
	switch {
	case v == nil:
		return "is required"
	case !validScore(*v, ScoreMin, ScoreMax):
		return "must be between 400 and 1600, in steps of 10"
	}
	return ""
}

// parseExamDate reads YYYY-MM-DD. nil or "" means "not decided yet". The date must be today or
// later (in the learner's time zone) and at most ExamDateMaxYears ahead.
func parseExamDate(raw *string, today time.Time) (*time.Time, string) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, ""
	}
	d, err := time.Parse(time.DateOnly, strings.TrimSpace(*raw))
	if err != nil {
		return nil, "must be a date in the format YYYY-MM-DD"
	}
	switch {
	case d.Before(today):
		return nil, "must be today or later"
	case d.After(today.AddDate(ExamDateMaxYears, 0, 0)):
		return nil, "must be within the next 2 years"
	}
	return &d, ""
}

// checkReason: optional; one of Reasons.
func checkReason(raw *string) (string, string) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return "", ""
	}
	r := strings.TrimSpace(*raw)
	if !slices.Contains(Reasons, r) {
		return "", "must be one of: " + strings.Join(Reasons, ", ")
	}
	return r, ""
}

func checkDailyMinutes(v *int) string {
	switch {
	case v == nil:
		return "is required"
	case !slices.Contains(DailyMinutes, *v):
		return "must be one of: 15, 30, 45, 60, 90, 120"
	}
	return ""
}

// normalizeStudyDays sorts ISO weekdays and drops repeats. At least one day is required.
func normalizeStudyDays(days []int) ([]int, string) {
	if len(days) == 0 {
		return nil, "choose at least one day"
	}
	if len(days) > 14 {
		return nil, "must list each weekday at most once"
	}
	out := make([]int, 0, len(days))
	for _, d := range days {
		if d < 1 || d > 7 {
			return nil, "must be weekdays from 1 (Monday) to 7 (Sunday)"
		}
		if !slices.Contains(out, d) {
			out = append(out, d)
		}
	}
	slices.Sort(out)
	return out, ""
}

func checkConfidence(v *int) string {
	switch {
	case v == nil:
		return "is required"
	case *v < ConfidenceMin || *v > ConfidenceMax:
		return "must be from 1 to 5"
	}
	return ""
}

// PreviousScoreInput is the raw earlier SAT result.
type PreviousScoreInput struct {
	Total          *int
	Math           *int
	ReadingWriting *int
	// TakenMonth is YYYY-MM, optional.
	TakenMonth *string
}

// checkPreviousScore validates an earlier result. Field errors are keyed under
// "previous_score.<field>". Sections are optional but come together and add up to the total.
func checkPreviousScore(in *PreviousScoreInput, today time.Time, fields map[string]string) *PreviousScore {
	if in == nil {
		return nil
	}
	ok := true
	fail := func(field, msg string) {
		fields["previous_score."+field] = msg
		ok = false
	}
	out := PreviousScore{}
	switch {
	case in.Total == nil:
		fail("total", "is required")
	case !validScore(*in.Total, ScoreMin, ScoreMax):
		fail("total", "must be between 400 and 1600, in steps of 10")
	default:
		out.Total = *in.Total
	}
	section := func(field string, v *int) int {
		if !validScore(*v, SectionMin, SectionMax) {
			fail(field, "must be between 200 and 800, in steps of 10")
			return 0
		}
		return *v
	}
	switch {
	case in.Math == nil && in.ReadingWriting == nil:
	case in.Math == nil:
		fail("math", "enter both section scores, or neither")
	case in.ReadingWriting == nil:
		fail("reading_writing", "enter both section scores, or neither")
	default:
		out.Math = section("math", in.Math)
		out.ReadingWriting = section("reading_writing", in.ReadingWriting)
		if out.Math != 0 && out.ReadingWriting != 0 && out.Total != 0 && out.Math+out.ReadingWriting != out.Total {
			fail("total", "must equal Math plus Reading and Writing")
		}
	}
	if in.TakenMonth != nil && strings.TrimSpace(*in.TakenMonth) != "" {
		m, err := time.Parse("2006-01", strings.TrimSpace(*in.TakenMonth))
		thisMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		switch {
		case err != nil:
			fail("taken_month", "must be a month in the format YYYY-MM")
		case m.Before(firstScoreMonth):
			fail("taken_month", "must be March 2016 or later (the 1600-point SAT)")
		case m.After(thisMonth):
			fail("taken_month", "can't be in the future")
		default:
			out.TakenOn = &m
		}
	}
	if !ok {
		return nil
	}
	return &out
}

// dayIn returns the calendar day of t in the time zone (UTC when it is unknown), at midnight UTC,
// so it compares with dates parsed by time.Parse.
func dayIn(t time.Time, tz string) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil || tz == "" {
		loc = time.UTC
	}
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}
