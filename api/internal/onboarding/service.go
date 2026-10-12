package onboarding

import (
	"context"
	"errors"
	"time"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
)

// View is the onboarding state as the API returns it.
type View struct {
	Record
	// Today is the learner's current day (their time zone), for days-until-exam.
	Today time.Time
}

// NextStep is the first step not saved yet, or "" when every step is saved.
func (v View) NextStep() Step {
	if m := v.Missing(); len(m) > 0 {
		return m[0]
	}
	return ""
}

// DaysUntilExam is nil when no exam date is set; 0 on the exam day; negative once it passed.
func (v View) DaysUntilExam() *int {
	if v.GoalSavedAt == nil || v.Goal.ExamDate == nil {
		return nil
	}
	d := int(v.Goal.ExamDate.Sub(v.Today).Hours() / 24)
	return &d
}

// GoalInput, ScheduleInput and BackgroundInput are the raw step forms; nil is "not sent".
type GoalInput struct {
	TargetScore *int
	ExamDate    *string
	Reason      *string
}

type ScheduleInput struct {
	DailyMinutes *int
	StudyDays    []int
}

type BackgroundInput struct {
	MathConfidence           *int
	ReadingWritingConfidence *int
	PreviousScore            *PreviousScoreInput
}

type ServiceDeps struct {
	Store    Store
	Profiles Profiles
	Now      func() time.Time
}

type Service struct {
	store    Store
	profiles Profiles
	now      func() time.Time
}

func NewService(d ServiceDeps) (*Service, error) {
	if d.Store == nil || d.Profiles == nil {
		return nil, errors.New("onboarding: Store and Profiles are required")
	}
	now := d.Now
	if now == nil {
		now = time.Now
	}
	return &Service{store: d.Store, profiles: d.Profiles, now: now}, nil
}

var (
	errAccountGone  = apperr.Unauthorized("Sign in to continue.")
	errProfileFirst = apperr.Conflict("Finish setting up your profile first.").
			WithDetails(map[string]any{"reason": "profile_setup_required"})
)

// Get returns the learner's onboarding state. Nothing saved yet is an empty state (every step
// nil), not an error, so readers such as the dashboard can tell "no goal set" apart.
func (s *Service) Get(ctx context.Context, userID string) (View, error) {
	st, err := s.profiles.Status(ctx, userID)
	if err != nil {
		return View{}, err
	}
	r, err := s.store.Get(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		r, err = Record{UserID: userID}, nil
	}
	if err != nil {
		return View{}, err
	}
	return View{Record: r, Today: dayIn(s.now(), st.Timezone)}, nil
}

// writable checks Profile Setup is done and returns the learner's today.
func (s *Service) writable(ctx context.Context, userID string) (time.Time, error) {
	st, err := s.profiles.Status(ctx, userID)
	if err != nil {
		return time.Time{}, err
	}
	if !st.Completed {
		return time.Time{}, errProfileFirst
	}
	return dayIn(s.now(), st.Timezone), nil
}

// SaveGoal validates and saves step 1. Validation errors come back all at once, per field.
func (s *Service) SaveGoal(ctx context.Context, userID string, in GoalInput) (View, error) {
	today, err := s.writable(ctx, userID)
	if err != nil {
		return View{}, err
	}
	fields := map[string]string{}
	var g Goal
	var msg string
	if msg = checkTargetScore(in.TargetScore); msg != "" {
		fields["target_score"] = msg
	} else {
		g.TargetScore = *in.TargetScore
	}
	if g.ExamDate, msg = parseExamDate(in.ExamDate, today); msg != "" {
		fields["exam_date"] = msg
	}
	if g.Reason, msg = checkReason(in.Reason); msg != "" {
		fields["reason"] = msg
	}
	if len(fields) > 0 {
		return View{}, validation(fields)
	}
	r, err := s.store.SaveGoal(ctx, userID, g, s.now())
	return written(r, err, today)
}

// SaveSchedule validates and saves step 2.
func (s *Service) SaveSchedule(ctx context.Context, userID string, in ScheduleInput) (View, error) {
	today, err := s.writable(ctx, userID)
	if err != nil {
		return View{}, err
	}
	fields := map[string]string{}
	var sc Schedule
	var msg string
	if msg = checkDailyMinutes(in.DailyMinutes); msg != "" {
		fields["daily_minutes"] = msg
	} else {
		sc.DailyMinutes = *in.DailyMinutes
	}
	if sc.StudyDays, msg = normalizeStudyDays(in.StudyDays); msg != "" {
		fields["study_days"] = msg
	}
	if len(fields) > 0 {
		return View{}, validation(fields)
	}
	r, err := s.store.SaveSchedule(ctx, userID, sc, s.now())
	return written(r, err, today)
}

// SaveBackground validates and saves step 3. The earlier SAT result is optional.
func (s *Service) SaveBackground(ctx context.Context, userID string, in BackgroundInput) (View, error) {
	today, err := s.writable(ctx, userID)
	if err != nil {
		return View{}, err
	}
	fields := map[string]string{}
	var b Background
	if msg := checkConfidence(in.MathConfidence); msg != "" {
		fields["math_confidence"] = msg
	} else {
		b.MathConfidence = *in.MathConfidence
	}
	if msg := checkConfidence(in.ReadingWritingConfidence); msg != "" {
		fields["reading_writing_confidence"] = msg
	} else {
		b.ReadingWritingConfidence = *in.ReadingWritingConfidence
	}
	b.Previous = checkPreviousScore(in.PreviousScore, today, fields)
	if len(fields) > 0 {
		return View{}, validation(fields)
	}
	r, err := s.store.SaveBackground(ctx, userID, b, s.now())
	return written(r, err, today)
}

// Complete ends onboarding once every step is saved. Calling it again changes nothing.
func (s *Service) Complete(ctx context.Context, userID string) (View, error) {
	today, err := s.writable(ctx, userID)
	if err != nil {
		return View{}, err
	}
	r, err := s.store.Complete(ctx, userID, s.now())
	if errors.Is(err, ErrIncomplete) || errors.Is(err, ErrNotFound) {
		missing := Steps
		if cur, gerr := s.store.Get(ctx, userID); gerr == nil {
			missing = cur.Missing()
		}
		names := make([]string, len(missing))
		for i, m := range missing {
			names[i] = string(m)
		}
		return View{}, apperr.Conflict("Answer every onboarding step before finishing.").
			WithDetails(map[string]any{"reason": "onboarding_incomplete", "missing_steps": names})
	}
	if err != nil {
		return View{}, err
	}
	return View{Record: r, Today: today}, nil
}

// written maps a store write to a view (ErrNotFound: the account was deleted meanwhile).
func written(r Record, err error, today time.Time) (View, error) {
	if errors.Is(err, ErrNotFound) {
		return View{}, errAccountGone
	}
	if err != nil {
		return View{}, err
	}
	return View{Record: r, Today: today}, nil
}

func validation(fields map[string]string) error {
	return apperr.New(apperr.CodeValidation, "Some fields are invalid").
		WithDetails(map[string]any{"fields": fields})
}
