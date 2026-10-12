package onboarding

import (
	"slices"
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }

var today = time.Date(2026, time.October, 13, 0, 0, 0, 0, time.UTC)

func TestCheckTargetScore(t *testing.T) {
	for _, v := range []int{400, 1050, 1600} {
		if msg := checkTargetScore(ptr(v)); msg != "" {
			t.Errorf("%d: %s", v, msg)
		}
	}
	for _, v := range []int{0, 390, 1610, 1405, -400} {
		if checkTargetScore(ptr(v)) == "" {
			t.Errorf("%d accepted", v)
		}
	}
	if checkTargetScore(nil) != "is required" {
		t.Error("nil accepted")
	}
}

func TestParseExamDate(t *testing.T) {
	if d, msg := parseExamDate(nil, today); d != nil || msg != "" {
		t.Fatal("nil must mean undecided")
	}
	if d, msg := parseExamDate(ptr("  "), today); d != nil || msg != "" {
		t.Fatal("blank must mean undecided")
	}
	if d, msg := parseExamDate(ptr("2026-10-13"), today); msg != "" || !d.Equal(today) {
		t.Fatal("today must be allowed:", msg)
	}
	if _, msg := parseExamDate(ptr("2028-10-13"), today); msg != "" {
		t.Fatal("exactly 2 years ahead:", msg)
	}
	for raw, want := range map[string]string{
		"2026-10-12": "must be today or later",
		"2028-10-14": "must be within the next 2 years",
		"13.10.2026": "must be a date in the format YYYY-MM-DD",
		"2026-02-30": "must be a date in the format YYYY-MM-DD",
	} {
		if _, msg := parseExamDate(ptr(raw), today); msg != want {
			t.Errorf("%s: %q", raw, msg)
		}
	}
}

func TestCheckReason(t *testing.T) {
	if r, msg := checkReason(ptr(" scholarship ")); r != "scholarship" || msg != "" {
		t.Fatal(r, msg)
	}
	if r, msg := checkReason(nil); r != "" || msg != "" {
		t.Fatal("optional")
	}
	if _, msg := checkReason(ptr("fun")); msg == "" {
		t.Fatal("unknown reason accepted")
	}
}

func TestScheduleChecks(t *testing.T) {
	if checkDailyMinutes(ptr(45)) != "" || checkDailyMinutes(ptr(40)) == "" || checkDailyMinutes(nil) == "" {
		t.Fatal("daily minutes")
	}
	days, msg := normalizeStudyDays([]int{7, 1, 3, 1})
	if msg != "" || !slices.Equal(days, []int{1, 3, 7}) {
		t.Fatal(days, msg)
	}
	for _, bad := range [][]int{nil, {}, {0}, {8}, {1, 2, 3, 4, 5, 6, 7, 1, 2, 3, 4, 5, 6, 7, 1}} {
		if _, msg := normalizeStudyDays(bad); msg == "" {
			t.Errorf("%v accepted", bad)
		}
	}
}

func TestCheckConfidence(t *testing.T) {
	if checkConfidence(ptr(1)) != "" || checkConfidence(ptr(5)) != "" {
		t.Fatal("range ends")
	}
	if checkConfidence(ptr(0)) == "" || checkConfidence(ptr(6)) == "" || checkConfidence(nil) == "" {
		t.Fatal("out of range accepted")
	}
}

func TestCheckPreviousScore(t *testing.T) {
	check := func(in *PreviousScoreInput) (*PreviousScore, map[string]string) {
		fields := map[string]string{}
		return checkPreviousScore(in, today, fields), fields
	}
	if p, f := check(nil); p != nil || len(f) > 0 {
		t.Fatal("nil means no earlier result")
	}
	p, f := check(&PreviousScoreInput{Total: ptr(1200)})
	if len(f) > 0 || p.Total != 1200 || p.Math != 0 || p.TakenOn != nil {
		t.Fatal(p, f)
	}
	p, f = check(&PreviousScoreInput{Total: ptr(1210), Math: ptr(650), ReadingWriting: ptr(560), TakenMonth: ptr("2026-10")})
	if len(f) > 0 || p.Math != 650 || p.ReadingWriting != 560 || !p.TakenOn.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal(p, f)
	}

	cases := []struct {
		in    PreviousScoreInput
		field string
	}{
		{PreviousScoreInput{}, "previous_score.total"},
		{PreviousScoreInput{Total: ptr(1700)}, "previous_score.total"},
		{PreviousScoreInput{Total: ptr(1200), Math: ptr(600)}, "previous_score.reading_writing"},
		{PreviousScoreInput{Total: ptr(1200), ReadingWriting: ptr(600)}, "previous_score.math"},
		{PreviousScoreInput{Total: ptr(1200), Math: ptr(900), ReadingWriting: ptr(300)}, "previous_score.math"},
		{PreviousScoreInput{Total: ptr(1200), Math: ptr(605), ReadingWriting: ptr(595)}, "previous_score.math"},
		{PreviousScoreInput{Total: ptr(1200), Math: ptr(600), ReadingWriting: ptr(500)}, "previous_score.total"},
		{PreviousScoreInput{Total: ptr(1200), TakenMonth: ptr("2026-11")}, "previous_score.taken_month"},
		{PreviousScoreInput{Total: ptr(1200), TakenMonth: ptr("2016-02")}, "previous_score.taken_month"},
		{PreviousScoreInput{Total: ptr(1200), TakenMonth: ptr("10/2025")}, "previous_score.taken_month"},
	}
	for _, c := range cases {
		p, f := check(&c.in)
		if p != nil || f[c.field] == "" {
			t.Errorf("%+v: %v %v", c.in, p, f)
		}
	}
}

func TestDayIn(t *testing.T) {
	if got := dayIn(fixedNow, "Asia/Tashkent"); !got.Equal(today) {
		t.Fatal(got)
	}
	if got := dayIn(fixedNow, ""); got.Day() != 12 {
		t.Fatal(got)
	}
	if got := dayIn(fixedNow, "Nowhere/City"); got.Day() != 12 {
		t.Fatal(got)
	}
}
