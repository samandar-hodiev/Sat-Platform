package onboarding

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/samandar-hodiev/sat-platform/api/internal/platform/migrate"
	"github.com/samandar-hodiev/sat-platform/api/migrations"
)

// TestPostgresStore runs the service and HTTP scenarios against PostgreSQL. It needs a
// disposable database (TEST_DATABASE_URL) and deletes every user in it between scenarios.
func TestPostgresStore(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	mg, err := migrate.New(url, migrations.FS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := mg.Up(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	_ = mg.Close()

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	saved := newStore
	newStore = func(t *testing.T, users ...string) Store {
		t.Helper()
		// DELETE, not TRUNCATE: audit_logs is append-only and only allows its FK to go NULL.
		if _, err := pool.Exec(ctx, "DELETE FROM users"); err != nil {
			t.Fatal(err)
		}
		for i, id := range users {
			if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, email_verified_at) VALUES ($1, $2, now())`,
				id, "learner"+string(rune('a'+i))+"@example.com"); err != nil {
				t.Fatal(err)
			}
		}
		return NewPostgresStore(pool)
	}
	t.Cleanup(func() { newStore = saved })

	scenarios := map[string]func(*testing.T){
		"fresh onboarding":           TestGetFreshOnboarding,
		"steps need profile setup":   TestStepsNeedProfileSetup,
		"full flow":                  TestFullOnboardingFlow,
		"complete with nothing":      TestCompleteWithNothingSaved,
		"previous score":             TestSaveBackgroundWithPreviousScore,
		"deleted account":            TestWriteForDeletedAccount,
		"http flow":                  TestHTTPOnboardingFlow,
		"http empty state":           TestHTTPEmptyStateIsNull,
		"http previous total only":   TestHTTPPreviousScoreTotalOnly,
		"concurrent complete":        testConcurrentComplete,
		"constraints reject garbage": func(t *testing.T) { testConstraints(t, pool) },
	}
	for name, fn := range scenarios {
		t.Run(name, fn)
	}
}

func saveAll(t *testing.T, f *fixture) {
	t.Helper()
	if _, err := f.svc.SaveGoal(ctx, userReady, validGoal()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.SaveSchedule(ctx, userReady, validSchedule()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.SaveBackground(ctx, userReady, validBackground()); err != nil {
		t.Fatal(err)
	}
}

func testConcurrentComplete(t *testing.T) {
	f := newFixture(t)
	saveAll(t, f)
	const n = 8
	times := make(chan time.Time, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			r, err := f.store.Complete(ctx, userReady, fixedNow.Add(time.Duration(i)*time.Second))
			if err != nil {
				t.Error(err)
			}
			times <- *r.CompletedAt
		}(i)
	}
	first := <-times
	for i := 1; i < n; i++ {
		if got := <-times; !got.Equal(first) {
			t.Fatalf("completed_at changed: %v vs %v", first, got)
		}
	}
}

// The CHECK constraints hold the same rules as validate.go, for writes from anywhere else.
func testConstraints(t *testing.T, pool *pgxpool.Pool) {
	newFixture(t)
	for name, stmt := range map[string]string{
		"target step":        `INSERT INTO learner_study_profiles (user_id, target_score) VALUES ($1, 1405)`,
		"goal without score": `INSERT INTO learner_study_profiles (user_id, goal_saved_at) VALUES ($1, now())`,
		"bad weekday":        `INSERT INTO learner_study_profiles (user_id, study_days) VALUES ($1, '{0}')`,
		"empty days":         `INSERT INTO learner_study_profiles (user_id, study_days) VALUES ($1, '{}')`,
		"bad minutes":        `INSERT INTO learner_study_profiles (user_id, daily_minutes) VALUES ($1, 50)`,
		"confidence":         `INSERT INTO learner_study_profiles (user_id, math_confidence) VALUES ($1, 6)`,
		"sections sum":       `INSERT INTO learner_study_profiles (user_id, prev_total_score, prev_math_score, prev_rw_score) VALUES ($1, 1200, 700, 600)`,
		"one section":        `INSERT INTO learner_study_profiles (user_id, prev_total_score, prev_math_score) VALUES ($1, 1200, 600)`,
		"taken mid-month":    `INSERT INTO learner_study_profiles (user_id, prev_total_score, prev_taken_on) VALUES ($1, 1200, '2025-05-15')`,
		"reason":             `INSERT INTO learner_study_profiles (user_id, reason) VALUES ($1, 'fun')`,
		"complete early":     `INSERT INTO learner_study_profiles (user_id, completed_at) VALUES ($1, now())`,
	} {
		if _, err := pool.Exec(ctx, stmt, userReady); err == nil {
			t.Errorf("%s: accepted", name)
			_, _ = pool.Exec(ctx, `DELETE FROM learner_study_profiles WHERE user_id = $1`, userReady)
		}
	}
	// Deleting the user removes the answers.
	f := newFixture(t)
	saveAll(t, f)
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userReady); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Get(ctx, userReady); err != ErrNotFound {
		t.Fatal(err)
	}
}
