package profile

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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
		"fresh profile":                   TestGetFreshProfile,
		"complete setup":                  TestCompleteSetupEmailAccount,
		"every invalid field":             TestCompleteSetupReportsEveryInvalidField,
		"phone optional":                  TestCompleteSetupPhoneOptional,
		"password policy rolls back":      TestCompleteSetupPasswordPolicyRollsBack,
		"password step error rolls back":  TestCompleteSetupPasswordStepErrorRollsBack,
		"google password optional":        TestCompleteSetupGooglePasswordOptional,
		"password exists":                 TestCompleteSetupRejectsPasswordWhenOneExists,
		"prefill from google":             TestPrefillFromGoogle,
		"prefill keeps completed profile": TestPrefillDoesNotTouchCompletedProfile,
		"avatar upload replace remove":    TestAvatarUploadReplaceRemove,
		"avatar kept by setup":            TestAvatarUploadKeepsSetupState,
		"http setup flow":                 TestHTTPSetupFlow,
		"http avatar upload serve remove": TestHTTPAvatarUploadServeRemove,
		"setup for deleted user":          testSetupForMissingUser,
		"concurrent setup completes once": testConcurrentSetup,
		"password runs in profile tx":     testPasswordSeesProfileTx,
	}
	for name, fn := range scenarios {
		t.Run(name, fn)
	}
}

// The account vanished between the auth check and the write (account deletion, 2.10).
func testSetupForMissingUser(t *testing.T) {
	f := newFixture(t)
	if _, err := f.store.(*PostgresStore).db.Exec(ctx, "DELETE FROM users WHERE id = $1", userEmail); err != nil {
		t.Fatal(err)
	}
	_, err := f.store.CompleteSetup(ctx, userEmail, Setup{FirstName: "A", LastName: "B", DateOfBirth: time.Date(2008, 1, 1, 0, 0, 0, 0, time.UTC)}, fixedNow, nil)
	if err != ErrNotFound {
		t.Fatal(err)
	}
	if _, err := f.store.SetAvatar(ctx, userEmail, "avatars/x/y.jpg"); err != ErrNotFound {
		t.Fatal(err)
	}
	if err := f.store.Prefill(ctx, userEmail, "A", "B"); err != ErrNotFound {
		t.Fatal(err)
	}
}

func testConcurrentSetup(t *testing.T) {
	f := newFixture(t)
	in := Setup{FirstName: "A", LastName: "B", DateOfBirth: time.Date(2008, 1, 1, 0, 0, 0, 0, time.UTC)}
	const n = 8
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			_, err := f.store.CompleteSetup(ctx, userGoogle, in, fixedNow, nil)
			errs <- err
		}()
	}
	ok := 0
	for i := 0; i < n; i++ {
		switch err := <-errs; err {
		case nil:
			ok++
		case ErrAlreadyCompleted:
		default:
			t.Fatal(err)
		}
	}
	if ok != 1 {
		t.Fatalf("%d submits completed the profile", ok)
	}
}

// The password callback runs in the same transaction as the profile write.
func testPasswordSeesProfileTx(t *testing.T) {
	f := newFixture(t)
	in := Setup{FirstName: "A", LastName: "B", DateOfBirth: time.Date(2008, 1, 1, 0, 0, 0, 0, time.UTC)}
	_, err := f.store.CompleteSetup(ctx, userEmail, in, fixedNow, func(tx pgx.Tx) error {
		var done bool
		if err := tx.QueryRow(ctx, `SELECT profile_completed_at IS NOT NULL FROM profiles WHERE user_id = $1`, userEmail).Scan(&done); err != nil {
			return err
		}
		if !done {
			t.Error("profile write not visible inside the transaction")
		}
		_, err := tx.Exec(ctx, `UPDATE users SET password_hash = 'x' WHERE id = $1`, userEmail)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var hash string
	if err := f.store.(*PostgresStore).db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userEmail).Scan(&hash); err != nil || hash != "x" {
		t.Fatal(hash, err)
	}
}
