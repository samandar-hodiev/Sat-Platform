package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/samandar-hodiev/sat-platform/api/internal/platform/migrate"
	"github.com/samandar-hodiev/sat-platform/api/migrations"
)

// TestPostgresStore runs the service scenarios against PostgreSQL. It needs a disposable
// database (TEST_DATABASE_URL) and deletes every user in it between scenarios.
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

	reset := func(t *testing.T) {
		t.Helper()
		// DELETE, not TRUNCATE: audit_logs is append-only and only allows its FK to go NULL.
		for _, q := range []string{"DELETE FROM users", "DELETE FROM email_verification_codes"} {
			if _, err := pool.Exec(context.Background(), q); err != nil {
				t.Fatal(err)
			}
		}
	}

	saved := newStore
	newStore = func(t *testing.T) Store {
		reset(t)
		return NewPostgresStore(pool)
	}
	t.Cleanup(func() { newStore = saved })

	scenarios := map[string]func(*testing.T){
		"sign-up creates verified learner": TestSignUpCreatesVerifiedLearner,
		"code single use":                  TestSignUpCodeIsSingleUse,
		"existing email signs in":          TestSignUpExistingEmailSignsIn,
		"wrong code counts attempts":       TestSignUpWrongCodeCountsAttempts,
		"code expires":                     TestSignUpCodeExpires,
		"resend cooldown and daily cap":    TestResendCooldownAndDailyCap,
		"sign-in":                          TestSignIn,
		"sign-in without password":         TestSignInWithoutPasswordLooksLikeWrongPassword,
		"rehash":                           TestSignInRehashesOldParams,
		"refresh rotation and reuse":       TestRefreshRotatesAndDetectsReuse,
		"refresh unknown and expired":      TestRefreshRejectsUnknownAndExpired,
		"logout one session":               TestLogoutRevokesOnlyThatSession,
		"password reset":                   TestPasswordResetFlow,
		"reset sets first password":        TestPasswordResetSetsFirstPasswordForCodeOnlyAccount,
		"reset expiry and supersede":       TestPasswordResetTokenExpiresAndIsSuperseded,
		"me":                               TestMe,
		"google":                           TestGoogleSignIn,
		"http flow":                        TestHTTPSignUpSignInRefreshMeLogout,
		// My Profile (2.10), account_test.go.
		"account: sign-in methods":        TestSignInMethods,
		"account: reauth password":        TestReauthWithPassword,
		"account: reauth code":            TestReauthWithCode,
		"account: codes bound to purpose": TestReauthCodeCannotBeUsedForOtherPurposes,
		"account: email change":           TestEmailChange,
		"account: email taken meanwhile":  TestEmailChangeTakenMeanwhileKeepsCode,
		"account: email change by code":   TestEmailChangeWithoutPasswordUsesCode,
		"account: change password":        TestChangePassword,
		"account: first password by code": TestSetFirstPasswordWithCode,
		"account: link and unlink google": TestLinkAndUnlinkGoogle,
		"account: delete":                 TestDeleteAccountTx,
		"account: needs active learner":   TestAccountChangesNeedActiveLearner,
		"account: http flow":              TestHTTPAccountFlow,
	}
	for name, run := range scenarios {
		t.Run(name, run)
	}

	t.Run("suspended and deleted accounts", func(t *testing.T) {
		f := newFixture(t)
		u := f.withPassword(t, "s@example.com", "correct-password")
		if _, err := pool.Exec(context.Background(), "UPDATE users SET status = 'suspended' WHERE id = $1", u.ID); err != nil {
			t.Fatal(err)
		}
		_, err := f.svc.SignIn(ctx, "s@example.com", "correct-password", true, web)
		wantCode(t, err, "FORBIDDEN")

		if _, err := pool.Exec(context.Background(), "UPDATE users SET status = 'active', deleted_at = now() WHERE id = $1", u.ID); err != nil {
			t.Fatal(err)
		}
		_, err = f.svc.SignIn(ctx, "s@example.com", "correct-password", true, web)
		wantCode(t, err, "UNAUTHORIZED")
		// The email is free again: signing up creates a new account.
		f.advance(2 * time.Minute)
		res := f.signUp(t, "s@example.com")
		if !res.IsNewUser || res.User.ID == u.ID {
			t.Fatalf("deleted account must not be reused: %+v", res)
		}
	})

	t.Run("account: deletion scrubs the users row (2.10)", func(t *testing.T) {
		f := newFixture(t)
		u := f.withPassword(t, "gone@example.com", "correct-password")
		if err := f.store.LinkIdentity(ctx, u.ID, ProviderGoogle, "sub-gone", "gone@gmail.com", f.now); err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.SendReauthCode(ctx, u.ID); err == nil {
			t.Fatal("password account must not get a reauth code")
		}
		if err := f.store.InTx(ctx, func(tx Tx) error { return f.svc.DeleteAccountTx(ctx, tx, u.ID) }); err != nil {
			t.Fatal(err)
		}
		var email string
		var hasPassword, verified, deleted bool
		var identities, live int
		if err := pool.QueryRow(context.Background(), `
			SELECT email, password_hash IS NOT NULL, email_verified_at IS NOT NULL, deleted_at IS NOT NULL,
			       (SELECT count(*) FROM user_identities WHERE user_id = $1),
			       (SELECT count(*) FROM auth_sessions WHERE user_id = $1 AND revoked_at IS NULL)
			  FROM users WHERE id = $1`, u.ID).Scan(&email, &hasPassword, &verified, &deleted, &identities, &live); err != nil {
			t.Fatal(err)
		}
		if email != "deleted-"+u.ID+"@"+DeletedEmailDomain || hasPassword || verified || !deleted || identities != 0 || live != 0 {
			t.Fatalf("email=%s password=%v verified=%v deleted=%v identities=%d live sessions=%d",
				email, hasPassword, verified, deleted, identities, live)
		}
	})

	t.Run("initial password joins the caller's transaction", func(t *testing.T) {
		f := newFixture(t)
		res := f.signUp(t, "tx@example.com")
		// The caller's transaction fails after the password write: nothing may be saved.
		err := pgx.BeginFunc(context.Background(), pool, func(tx pgx.Tx) error {
			if err := f.svc.SetInitialPasswordTx(ctx, PgxTx(tx), res.User.ID, "first-password"); err != nil {
				return err
			}
			return errors.New("profile upsert failed")
		})
		if err == nil {
			t.Fatal("want the caller's error")
		}
		u, _ := f.store.UserByID(ctx, res.User.ID)
		if u.PasswordHash != "" {
			t.Fatal("password must roll back with the caller's transaction")
		}
		err = pgx.BeginFunc(context.Background(), pool, func(tx pgx.Tx) error {
			return f.svc.SetInitialPasswordTx(ctx, PgxTx(tx), res.User.ID, "first-password")
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.SignIn(ctx, "tx@example.com", "first-password", true, web); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("session client info is stored", func(t *testing.T) {
		f := newFixture(t)
		res, err := func() (SessionResult, error) {
			_, _ = f.svc.StartSignUp(ctx, "c@example.com")
			return f.svc.VerifySignUp(ctx, "c@example.com", f.mail.code(t),
				ClientInfo{Platform: "ios", UserAgent: "SATApp/1.0", IP: "203.0.113.7"})
		}()
		if err != nil {
			t.Fatal(err)
		}
		var platform, ua, ip string
		err = pool.QueryRow(context.Background(),
			`SELECT s.platform, s.user_agent, host(s.ip_address) FROM auth_sessions s WHERE s.user_id = $1`, res.User.ID).
			Scan(&platform, &ua, &ip)
		if err != nil || platform != "ios" || ua != "SATApp/1.0" || ip != "203.0.113.7" {
			t.Fatalf("session row: %s %s %s %v", platform, ua, ip, err)
		}
	})
}
