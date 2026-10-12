package onboarding

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound: the learner has no onboarding row yet, or (on writes) no longer exists.
	ErrNotFound = errors.New("onboarding: not found")
	// ErrIncomplete: Complete was called before every step was saved.
	ErrIncomplete = errors.New("onboarding: steps missing")
)

// Step names, in the order the learner answers them.
type Step string

const (
	StepGoal       Step = "goal"
	StepSchedule   Step = "schedule"
	StepBackground Step = "background"
)

var Steps = []Step{StepGoal, StepSchedule, StepBackground}

// Goal is step 1 (PRD §8.1).
type Goal struct {
	TargetScore int
	// ExamDate nil: the learner has not decided yet.
	ExamDate *time.Time
	// Reason is one of Reasons, or "" when skipped.
	Reason string
}

// Schedule is step 2 (PRD §8.2).
type Schedule struct {
	DailyMinutes int
	// StudyDays are ISO weekdays (1 = Monday … 7 = Sunday), sorted, without repeats.
	StudyDays []int
}

// PreviousScore is an earlier official SAT result. Math and ReadingWriting are 0 when not given.
type PreviousScore struct {
	Total          int
	Math           int
	ReadingWriting int
	// TakenOn is the first day of the month the test was taken, nil when not given.
	TakenOn *time.Time
}

// Background is step 3 (PRD §8.2).
type Background struct {
	MathConfidence           int
	ReadingWritingConfidence int
	// Previous is nil when the learner has no earlier result (or skipped the question).
	Previous *PreviousScore
}

// Record is one learner_study_profiles row. A step's fields are meaningful only when its
// *SavedAt is set.
type Record struct {
	UserID            string
	Goal              Goal
	GoalSavedAt       *time.Time
	Schedule          Schedule
	ScheduleSavedAt   *time.Time
	Background        Background
	BackgroundSavedAt *time.Time
	CompletedAt       *time.Time
	UpdatedAt         time.Time
}

// Missing returns the steps not saved yet, in order.
func (r Record) Missing() []Step {
	var out []Step
	if r.GoalSavedAt == nil {
		out = append(out, StepGoal)
	}
	if r.ScheduleSavedAt == nil {
		out = append(out, StepSchedule)
	}
	if r.BackgroundSavedAt == nil {
		out = append(out, StepBackground)
	}
	return out
}

// Store persists onboarding answers (PostgreSQL in production, memory in unit tests).
type Store interface {
	// Get returns the learner's answers, or ErrNotFound when nothing is saved yet.
	Get(ctx context.Context, userID string) (Record, error)
	// SaveGoal, SaveSchedule and SaveBackground write one step (creating the row if needed) and
	// set its saved-at time. ErrNotFound when the user does not exist.
	SaveGoal(ctx context.Context, userID string, g Goal, now time.Time) (Record, error)
	SaveSchedule(ctx context.Context, userID string, s Schedule, now time.Time) (Record, error)
	SaveBackground(ctx context.Context, userID string, b Background, now time.Time) (Record, error)
	// Complete sets completed_at when every step is saved; a second call keeps the first time.
	// ErrIncomplete when a step is missing, ErrNotFound when there is no row.
	Complete(ctx context.Context, userID string, now time.Time) (Record, error)
}

// ProfileStatus is what onboarding needs from the learner's profile.
type ProfileStatus struct {
	// Completed: Profile Setup is finished (PRD §7: onboarding starts after it).
	Completed bool
	// Timezone is an IANA name; it decides which day is "today" for the exam date.
	Timezone string
}

// Profiles is the profile module's side (an adapter over profile.Service, wired in
// internal/server).
type Profiles interface {
	// Status returns API errors (e.g. UNAUTHORIZED when the account is gone) as they are.
	Status(ctx context.Context, userID string) (ProfileStatus, error)
}
