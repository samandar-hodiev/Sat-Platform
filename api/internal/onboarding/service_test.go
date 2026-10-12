package onboarding

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
)

var ctx = context.Background()

func wantCode(t *testing.T, err error, code apperr.Code) *apperr.Error {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != code {
		t.Fatalf("got %v, want %s", err, code)
	}
	return ae
}

func wantFields(t *testing.T, err error, fields ...string) {
	t.Helper()
	ae := wantCode(t, err, apperr.CodeValidation)
	got, _ := ae.Details["fields"].(map[string]string)
	if len(got) != len(fields) {
		t.Fatalf("fields %v, want %v", got, fields)
	}
	for _, f := range fields {
		if got[f] == "" {
			t.Fatalf("fields %v, missing %q", got, f)
		}
	}
}

func validGoal() GoalInput {
	return GoalInput{TargetScore: ptr(1450), ExamDate: ptr("2027-03-13"), Reason: ptr("college_admission")}
}

func validSchedule() ScheduleInput {
	return ScheduleInput{DailyMinutes: ptr(45), StudyDays: []int{6, 1, 3}}
}

func validBackground() BackgroundInput {
	return BackgroundInput{MathConfidence: ptr(3), ReadingWritingConfidence: ptr(4)}
}

func TestGetFreshOnboarding(t *testing.T) {
	f := newFixture(t)
	v, err := f.svc.Get(ctx, userReady)
	if err != nil {
		t.Fatal(err)
	}
	if v.CompletedAt != nil || v.GoalSavedAt != nil || v.NextStep() != StepGoal || v.DaysUntilExam() != nil {
		t.Fatalf("%+v", v)
	}
	// Reading works before Profile Setup is done (nothing is saved yet).
	if _, err := f.svc.Get(ctx, userNew); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Get(ctx, userGone)
	wantCode(t, err, apperr.CodeUnauthorized)
}

func TestStepsNeedProfileSetup(t *testing.T) {
	f := newFixture(t)
	for name, call := range map[string]func() error{
		"goal":       func() error { _, err := f.svc.SaveGoal(ctx, userNew, validGoal()); return err },
		"schedule":   func() error { _, err := f.svc.SaveSchedule(ctx, userNew, validSchedule()); return err },
		"background": func() error { _, err := f.svc.SaveBackground(ctx, userNew, validBackground()); return err },
		"complete":   func() error { _, err := f.svc.Complete(ctx, userNew); return err },
	} {
		ae := wantCode(t, call(), apperr.CodeConflict)
		if ae.Details["reason"] != "profile_setup_required" {
			t.Fatal(name, ae.Details)
		}
	}
	if _, err := f.store.Get(ctx, userNew); !errors.Is(err, ErrNotFound) {
		t.Fatal("nothing must be written", err)
	}
}

func TestFullOnboardingFlow(t *testing.T) {
	f := newFixture(t)
	v, err := f.svc.SaveGoal(ctx, userReady, validGoal())
	if err != nil {
		t.Fatal(err)
	}
	// Tashkent "today" is 13 Oct 2026; 13 Mar 2027 is 151 days later.
	if v.Goal.TargetScore != 1450 || v.Goal.Reason != "college_admission" || *v.DaysUntilExam() != 151 || v.NextStep() != StepSchedule {
		t.Fatalf("%+v days=%v", v.Goal, v.DaysUntilExam())
	}

	_, err = f.svc.Complete(ctx, userReady)
	ae := wantCode(t, err, apperr.CodeConflict)
	if ae.Details["reason"] != "onboarding_incomplete" ||
		!slices.Equal(ae.Details["missing_steps"].([]string), []string{"schedule", "background"}) {
		t.Fatal(ae.Details)
	}

	if v, err = f.svc.SaveSchedule(ctx, userReady, validSchedule()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(v.Schedule.StudyDays, []int{1, 3, 6}) || v.NextStep() != StepBackground {
		t.Fatalf("%+v", v.Schedule)
	}
	if v, err = f.svc.SaveBackground(ctx, userReady, validBackground()); err != nil {
		t.Fatal(err)
	}
	if v.Background.Previous != nil || v.NextStep() != "" || v.CompletedAt != nil {
		t.Fatalf("%+v", v.Background)
	}

	done, err := f.svc.Complete(ctx, userReady)
	if err != nil || done.CompletedAt == nil {
		t.Fatal(done, err)
	}
	// Idempotent, and answers stay editable after completion (Learning Preferences, 2.11).
	again, err := f.svc.Complete(ctx, userReady)
	if err != nil || !again.CompletedAt.Equal(*done.CompletedAt) {
		t.Fatal(again, err)
	}
	edited, err := f.svc.SaveGoal(ctx, userReady, GoalInput{TargetScore: ptr(1500)})
	if err != nil || edited.Goal.TargetScore != 1500 || edited.Goal.ExamDate != nil || edited.Goal.Reason != "" ||
		edited.CompletedAt == nil || edited.ScheduleSavedAt == nil {
		t.Fatalf("%+v %v", edited, err)
	}
	if edited.DaysUntilExam() != nil {
		t.Fatal("undecided exam date has no countdown")
	}
}

func TestCompleteWithNothingSaved(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Complete(ctx, userReady)
	ae := wantCode(t, err, apperr.CodeConflict)
	if len(ae.Details["missing_steps"].([]string)) != 3 {
		t.Fatal(ae.Details)
	}
}

func TestSaveGoalReportsEveryInvalidField(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.SaveGoal(ctx, userReady, GoalInput{ExamDate: ptr("2026-10-12"), Reason: ptr("x")})
	wantFields(t, err, "target_score", "exam_date", "reason")
	// 12 Oct is still "today" in UTC, but the learner is in Tashkent, where it is 13 Oct.
	if _, err := f.svc.SaveGoal(ctx, userReady, GoalInput{TargetScore: ptr(1200), ExamDate: ptr("2026-10-13")}); err != nil {
		t.Fatal(err)
	}
}

func TestSaveScheduleValidation(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.SaveSchedule(ctx, userReady, ScheduleInput{DailyMinutes: ptr(50), StudyDays: []int{9}})
	wantFields(t, err, "daily_minutes", "study_days")
	_, err = f.svc.SaveSchedule(ctx, userReady, ScheduleInput{})
	wantFields(t, err, "daily_minutes", "study_days")
}

func TestSaveBackgroundWithPreviousScore(t *testing.T) {
	f := newFixture(t)
	in := validBackground()
	in.PreviousScore = &PreviousScoreInput{Total: ptr(1180), Math: ptr(600), ReadingWriting: ptr(580), TakenMonth: ptr("2026-05")}
	v, err := f.svc.SaveBackground(ctx, userReady, in)
	if err != nil {
		t.Fatal(err)
	}
	p := v.Background.Previous
	if p == nil || p.Total != 1180 || p.Math != 600 || p.ReadingWriting != 580 || p.TakenOn.Month() != 5 {
		t.Fatalf("%+v", p)
	}
	// Saving again without it (Skip) clears it.
	if v, err = f.svc.SaveBackground(ctx, userReady, validBackground()); err != nil || v.Background.Previous != nil {
		t.Fatal(v.Background, err)
	}

	_, err = f.svc.SaveBackground(ctx, userReady, BackgroundInput{
		MathConfidence: ptr(0),
		PreviousScore:  &PreviousScoreInput{Total: ptr(1200), Math: ptr(600)},
	})
	wantFields(t, err, "math_confidence", "reading_writing_confidence", "previous_score.reading_writing")
}

func TestWriteForDeletedAccount(t *testing.T) {
	f := newFixture(t)
	f.svc.profiles = fakeProfiles{userGone: {Completed: true, Timezone: "UTC"}}
	_, err := f.svc.SaveGoal(ctx, userGone, validGoal())
	wantCode(t, err, apperr.CodeUnauthorized)
}
