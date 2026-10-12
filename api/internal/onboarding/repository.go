package onboarding

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements Store on the learner_study_profiles table.
type PostgresStore struct {
	db *pgxpool.Pool
}

func NewPostgresStore(db *pgxpool.Pool) *PostgresStore { return &PostgresStore{db: db} }

const recordColumns = `user_id,
	COALESCE(target_score, 0), exam_date, COALESCE(reason, ''), goal_saved_at,
	COALESCE(daily_minutes, 0), COALESCE(study_days, '{}'), schedule_saved_at,
	COALESCE(math_confidence, 0), COALESCE(rw_confidence, 0),
	prev_total_score, COALESCE(prev_math_score, 0), COALESCE(prev_rw_score, 0), prev_taken_on,
	background_saved_at, completed_at, updated_at`

func scanRecord(row pgx.Row) (Record, error) {
	var r Record
	var days []int16
	var prevTotal *int16
	var prevMath, prevRW int16
	var target, minutes, mathConf, rwConf int16
	var prevTaken *time.Time
	err := row.Scan(&r.UserID,
		&target, &r.Goal.ExamDate, &r.Goal.Reason, &r.GoalSavedAt,
		&minutes, &days, &r.ScheduleSavedAt,
		&mathConf, &rwConf, &prevTotal, &prevMath, &prevRW, &prevTaken,
		&r.BackgroundSavedAt, &r.CompletedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, err
	}
	r.Goal.TargetScore, r.Schedule.DailyMinutes = int(target), int(minutes)
	if len(days) > 0 {
		r.Schedule.StudyDays = make([]int, len(days))
		for i, d := range days {
			r.Schedule.StudyDays[i] = int(d)
		}
	}
	r.Background.MathConfidence, r.Background.ReadingWritingConfidence = int(mathConf), int(rwConf)
	if prevTotal != nil {
		r.Background.Previous = &PreviousScore{
			Total: int(*prevTotal), Math: int(prevMath), ReadingWriting: int(prevRW), TakenOn: prevTaken,
		}
	}
	return r, nil
}

func (s *PostgresStore) Get(ctx context.Context, userID string) (Record, error) {
	r, err := scanRecord(s.db.QueryRow(ctx, `SELECT `+recordColumns+` FROM learner_study_profiles WHERE user_id = $1`, userID))
	return r, mapErr(err)
}

func nullable[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

func (s *PostgresStore) SaveGoal(ctx context.Context, userID string, g Goal, now time.Time) (Record, error) {
	r, err := scanRecord(s.db.QueryRow(ctx, `
		INSERT INTO learner_study_profiles (user_id, target_score, exam_date, reason, goal_saved_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			target_score  = EXCLUDED.target_score,
			exam_date     = EXCLUDED.exam_date,
			reason        = EXCLUDED.reason,
			goal_saved_at = EXCLUDED.goal_saved_at
		RETURNING `+recordColumns,
		userID, g.TargetScore, g.ExamDate, nullable(g.Reason), now))
	return r, mapErr(err)
}

func (s *PostgresStore) SaveSchedule(ctx context.Context, userID string, sc Schedule, now time.Time) (Record, error) {
	days := make([]int16, len(sc.StudyDays))
	for i, d := range sc.StudyDays {
		days[i] = int16(d)
	}
	r, err := scanRecord(s.db.QueryRow(ctx, `
		INSERT INTO learner_study_profiles (user_id, daily_minutes, study_days, schedule_saved_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			daily_minutes     = EXCLUDED.daily_minutes,
			study_days        = EXCLUDED.study_days,
			schedule_saved_at = EXCLUDED.schedule_saved_at
		RETURNING `+recordColumns,
		userID, sc.DailyMinutes, days, now))
	return r, mapErr(err)
}

func (s *PostgresStore) SaveBackground(ctx context.Context, userID string, b Background, now time.Time) (Record, error) {
	var total, math, rw *int
	var taken *time.Time
	if p := b.Previous; p != nil {
		total, math, rw, taken = &p.Total, nullable(p.Math), nullable(p.ReadingWriting), p.TakenOn
	}
	r, err := scanRecord(s.db.QueryRow(ctx, `
		INSERT INTO learner_study_profiles (user_id, math_confidence, rw_confidence,
			prev_total_score, prev_math_score, prev_rw_score, prev_taken_on, background_saved_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			math_confidence     = EXCLUDED.math_confidence,
			rw_confidence       = EXCLUDED.rw_confidence,
			prev_total_score    = EXCLUDED.prev_total_score,
			prev_math_score     = EXCLUDED.prev_math_score,
			prev_rw_score       = EXCLUDED.prev_rw_score,
			prev_taken_on       = EXCLUDED.prev_taken_on,
			background_saved_at = EXCLUDED.background_saved_at
		RETURNING `+recordColumns,
		userID, b.MathConfidence, b.ReadingWritingConfidence, total, math, rw, taken, now))
	return r, mapErr(err)
}

// Complete: the WHERE makes one statement decide, so concurrent calls keep the first time.
func (s *PostgresStore) Complete(ctx context.Context, userID string, now time.Time) (Record, error) {
	r, err := scanRecord(s.db.QueryRow(ctx, `
		UPDATE learner_study_profiles SET completed_at = COALESCE(completed_at, $2)
		WHERE user_id = $1 AND goal_saved_at IS NOT NULL
			AND schedule_saved_at IS NOT NULL AND background_saved_at IS NOT NULL
		RETURNING `+recordColumns, userID, now))
	if errors.Is(err, ErrNotFound) {
		if _, gerr := s.Get(ctx, userID); gerr == nil {
			return Record{}, ErrIncomplete
		}
	}
	return r, mapErr(err)
}

// mapErr turns "the user does not exist" (FK violation, invalid UUID) into ErrNotFound.
func mapErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02") {
		return ErrNotFound
	}
	return err
}
