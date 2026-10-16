package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
)

// My Profile (2.10) account changes. The same scenarios run on PostgreSQL in
// account_integration_test.go.

// learnerSession signs a learner up and returns the user and the session id of that sign-in.
func (f *fixture) learnerSession(t *testing.T, email string) (User, string) {
	t.Helper()
	res := f.signUp(t, email)
	claims, err := f.svc.tokens.Verify(res.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	return res.User, claims.SessionID
}

// passwordSession: a learner with a password, signed in twice (two devices).
func (f *fixture) passwordSession(t *testing.T, email, password string) (user User, current, other string) {
	t.Helper()
	user = f.withPassword(t, email, password)
	for _, sid := range []*string{&current, &other} {
		res, err := f.svc.SignIn(ctx, email, password, true, web)
		if err != nil {
			t.Fatal(err)
		}
		claims, _ := f.svc.tokens.Verify(res.AccessToken)
		*sid = claims.SessionID
	}
	return user, current, other
}

func (f *fixture) revoked(t *testing.T, sessionID string) bool {
	t.Helper()
	s, err := f.store.SessionByID(ctx, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	return s.RevokedAt != nil
}

func (f *fixture) reauthCode(t *testing.T, userID string) string {
	t.Helper()
	f.advance(2 * time.Minute) // past the resend cooldown of earlier codes
	if _, err := f.svc.SendReauthCode(ctx, userID); err != nil {
		t.Fatal(err)
	}
	return f.mail.code(t)
}

func otherCode(code string) string {
	if code == "000000" {
		return "111111"
	}
	return "000000"
}

func TestSignInMethods(t *testing.T) {
	f := newFixture(t)
	u, _ := f.learnerSession(t, "a@example.com")
	m, err := f.svc.SignInMethods(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Email != "a@example.com" || !m.EmailVerified || m.HasPassword || m.Google != nil || m.ReauthMethod() != "email_code" {
		t.Fatalf("methods = %+v", m)
	}
	pw := f.withPassword(t, "b@example.com", "correct-password")
	m, _ = f.svc.SignInMethods(ctx, pw.ID)
	if !m.HasPassword || m.ReauthMethod() != "password" {
		t.Fatalf("methods = %+v", m)
	}
}

func TestReauthWithPassword(t *testing.T) {
	f := newFixture(t)
	u := f.withPassword(t, "a@example.com", "correct-password")

	wantField(t, f.svc.Reauthenticate(ctx, u.ID, Reauth{}), "current_password", "Enter your current password")
	wantField(t, f.svc.Reauthenticate(ctx, u.ID, Reauth{Password: "wrong-password"}), "current_password", "incorrect")
	if err := f.svc.Reauthenticate(ctx, u.ID, Reauth{Password: "correct-password"}); err != nil {
		t.Fatal(err)
	}
	// A code is not accepted instead of the password, and none can be requested.
	_, err := f.svc.SendReauthCode(ctx, u.ID)
	wantCode(t, err, apperr.CodeConflict)

	// The per-user budget stops guessing (5 per 15 minutes, the two above included).
	for range 3 {
		_ = f.svc.Reauthenticate(ctx, u.ID, Reauth{Password: "wrong-password"})
	}
	wantCode(t, f.svc.Reauthenticate(ctx, u.ID, Reauth{Password: "correct-password"}), apperr.CodeRateLimited)
}

func TestReauthWithCode(t *testing.T) {
	f := newFixture(t)
	u, _ := f.learnerSession(t, "a@example.com")

	wantField(t, f.svc.Reauthenticate(ctx, u.ID, Reauth{}), "code", "Enter the code")
	wantField(t, f.svc.Reauthenticate(ctx, u.ID, Reauth{Code: "123456"}), "code", "expired") // none sent

	sent, err := f.svc.SendReauthCode(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if msg := f.mail.last(t); msg.To != "a@example.com" || !strings.Contains(msg.Subject, "confirmation code") {
		t.Fatalf("mail = %+v", msg)
	}
	if !sent.ResendAvailableAt.Equal(f.now.Add(time.Minute)) {
		t.Fatalf("sent = %+v", sent)
	}
	code := f.mail.code(t)
	// Resend cooldown.
	_, err = f.svc.SendReauthCode(ctx, u.ID)
	wantCode(t, err, apperr.CodeRateLimited)

	wantField(t, f.svc.Reauthenticate(ctx, u.ID, Reauth{Code: otherCode(code)}), "code", "4 attempts left")
	if err := f.svc.Reauthenticate(ctx, u.ID, Reauth{Code: code}); err != nil {
		t.Fatal(err)
	}
	// Single use.
	wantField(t, f.svc.Reauthenticate(ctx, u.ID, Reauth{Code: code}), "code", "expired")

	// Expiry.
	code = f.reauthCode(t, u.ID)
	f.advance(11 * time.Minute)
	wantField(t, f.svc.Reauthenticate(ctx, u.ID, Reauth{Code: code}), "code", "expired")

	// Five wrong guesses lock the code, even the right one fails afterwards.
	code = f.reauthCode(t, u.ID)
	for range 4 {
		_ = f.svc.Reauthenticate(ctx, u.ID, Reauth{Code: otherCode(code)})
	}
	wantField(t, f.svc.Reauthenticate(ctx, u.ID, Reauth{Code: otherCode(code)}), "code", "Too many wrong attempts")
	wantField(t, f.svc.Reauthenticate(ctx, u.ID, Reauth{Code: code}), "code", "Too many wrong attempts")
}

func TestReauthCodeCannotBeUsedForOtherPurposes(t *testing.T) {
	f := newFixture(t)
	u, sid := f.learnerSession(t, "a@example.com")
	code := f.reauthCode(t, u.ID)
	// A reauth code is not a sign-up code for the same address…
	_, err := f.svc.VerifySignUp(ctx, "a@example.com", code, web)
	wantField(t, err, "code", "")
	// …nor an email-change code.
	_, err = f.svc.ConfirmEmailChange(ctx, u.ID, sid, code)
	wantField(t, err, "code", "expired")
}

func TestEmailChange(t *testing.T) {
	f := newFixture(t)
	u, current, other := f.passwordSession(t, "old@example.com", "correct-password")

	_, err := f.svc.StartEmailChange(ctx, u.ID, "not-an-email", Reauth{Password: "correct-password"})
	wantField(t, err, "new_email", "valid email")
	_, err = f.svc.StartEmailChange(ctx, u.ID, " OLD@example.com", Reauth{Password: "correct-password"})
	wantField(t, err, "new_email", "already your email")
	_, err = f.svc.StartEmailChange(ctx, u.ID, "new@example.com", Reauth{Password: "wrong-password"})
	wantField(t, err, "current_password", "incorrect")

	// Another account's address is refused, but only after the learner is confirmed.
	f.signUp(t, "taken@example.com")
	_, err = f.svc.StartEmailChange(ctx, u.ID, "taken@example.com", Reauth{Password: "correct-password"})
	wantField(t, err, "new_email", "already used")

	started, err := f.svc.StartEmailChange(ctx, u.ID, " New@Example.com ", Reauth{Password: "correct-password"})
	if err != nil {
		t.Fatal(err)
	}
	if started.Email != "new@example.com" || f.mail.last(t).To != "new@example.com" {
		t.Fatalf("started = %+v, mail to %s", started, f.mail.last(t).To)
	}
	code := f.mail.code(t)
	_, err = f.svc.ConfirmEmailChange(ctx, u.ID, current, otherCode(code))
	wantField(t, err, "code", "incorrect")

	m, err := f.svc.ConfirmEmailChange(ctx, u.ID, current, code)
	if err != nil {
		t.Fatal(err)
	}
	if m.Email != "new@example.com" || !m.EmailVerified || !m.HasPassword {
		t.Fatalf("methods = %+v", m)
	}
	notice := f.mail.last(t)
	if notice.To != "old@example.com" || !strings.Contains(notice.Text, "new@example.com") {
		t.Fatalf("notice = %+v", notice)
	}
	if f.revoked(t, current) || !f.revoked(t, other) {
		t.Fatal("the current session stays, others are signed out")
	}
	// Sign-in works with the new address only; the old one is free.
	if _, err := f.svc.SignIn(ctx, "new@example.com", "correct-password", false, web); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.SignIn(ctx, "old@example.com", "correct-password", false, web)
	wantCode(t, err, apperr.CodeUnauthorized)
	// The code is spent.
	_, err = f.svc.ConfirmEmailChange(ctx, u.ID, current, code)
	wantField(t, err, "code", "expired")
}

func TestEmailChangeTakenMeanwhileKeepsCode(t *testing.T) {
	f := newFixture(t)
	u, sid := f.learnerSession(t, "a@example.com")
	code := f.reauthCode(t, u.ID)
	if _, err := f.svc.StartEmailChange(ctx, u.ID, "b@example.com", Reauth{Code: code}); err != nil {
		t.Fatal(err)
	}
	changeCode := f.mail.code(t)
	// Someone signs up with the address before the learner confirms.
	f.signUp(t, "b@example.com")
	_, err := f.svc.ConfirmEmailChange(ctx, u.ID, sid, changeCode)
	wantField(t, err, "new_email", "already used")
	m, _ := f.svc.SignInMethods(ctx, u.ID)
	if m.Email != "a@example.com" {
		t.Fatalf("email = %s", m.Email)
	}
}

func TestEmailChangeWithoutPasswordUsesCode(t *testing.T) {
	f := newFixture(t)
	u, sid := f.learnerSession(t, "a@example.com")
	_, err := f.svc.StartEmailChange(ctx, u.ID, "b@example.com", Reauth{Password: "anything"})
	wantField(t, err, "code", "Enter the code")

	code := f.reauthCode(t, u.ID)
	if _, err := f.svc.StartEmailChange(ctx, u.ID, "b@example.com", Reauth{Code: code}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.ConfirmEmailChange(ctx, u.ID, sid, f.mail.code(t)); err != nil {
		t.Fatal(err)
	}
	// Learners without a password can still sign in with a code at the new address.
	f.advance(2 * time.Minute)
	res := f.signUp(t, "b@example.com")
	if res.IsNewUser || res.User.ID != u.ID {
		t.Fatalf("res = %+v", res)
	}
}

func TestChangePassword(t *testing.T) {
	f := newFixture(t)
	u, current, other := f.passwordSession(t, "a@example.com", "correct-password")

	wantField(t, f.svc.ChangePassword(ctx, u.ID, current, Reauth{Password: "correct-password"}, "short"), "new_password", "at least")
	wantField(t, f.svc.ChangePassword(ctx, u.ID, current, Reauth{Password: "wrong-password"}, "brand-new-password"), "current_password", "incorrect")
	wantField(t, f.svc.ChangePassword(ctx, u.ID, current, Reauth{Password: "correct-password"}, "correct-password"), "new_password", "different")

	if err := f.svc.ChangePassword(ctx, u.ID, current, Reauth{Password: "correct-password"}, "brand-new-password"); err != nil {
		t.Fatal(err)
	}
	if f.revoked(t, current) || !f.revoked(t, other) {
		t.Fatal("the current session stays, others are signed out")
	}
	if _, err := f.svc.SignIn(ctx, "a@example.com", "brand-new-password", false, web); err != nil {
		t.Fatal(err)
	}
	_, err := f.svc.SignIn(ctx, "a@example.com", "correct-password", false, web)
	wantCode(t, err, apperr.CodeUnauthorized)
}

func TestSetFirstPasswordWithCode(t *testing.T) {
	f := newFixture(t)
	u, sid := f.learnerSession(t, "a@example.com")
	wantField(t, f.svc.ChangePassword(ctx, u.ID, sid, Reauth{}, "first-password"), "code", "Enter the code")
	code := f.reauthCode(t, u.ID)
	if err := f.svc.ChangePassword(ctx, u.ID, sid, Reauth{Code: code}, "first-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.SignIn(ctx, "a@example.com", "first-password", false, web); err != nil {
		t.Fatal(err)
	}
	m, _ := f.svc.SignInMethods(ctx, u.ID)
	if !m.HasPassword || m.ReauthMethod() != "password" {
		t.Fatalf("methods = %+v", m)
	}
}

func TestLinkAndUnlinkGoogle(t *testing.T) {
	g := newFakeGoogle(t)
	f := newFixture(t)
	u := f.withPassword(t, "a@example.com", "correct-password")
	token := func(sub, email string) string {
		c := g.claims(f.now)
		c["sub"], c["email"] = sub, email
		return g.sign(c, g.kid, g.key)
	}

	// Disabled.
	_, err := f.svc.LinkGoogle(ctx, u.ID, token("g-a", "a@gmail.com"), Reauth{Password: "correct-password"})
	wantCode(t, err, apperr.CodeUnavailable)
	f.svc.google = g.verifier(f.now)

	_, err = f.svc.LinkGoogle(ctx, u.ID, "garbage", Reauth{Password: "correct-password"})
	wantCode(t, err, apperr.CodeBadRequest)
	_, err = f.svc.LinkGoogle(ctx, u.ID, token("g-a", "a@gmail.com"), Reauth{Password: "wrong-password"})
	wantField(t, err, "current_password", "incorrect")

	// A Google account that already signs in to another learner is refused.
	other, err := f.svc.SignInWithGoogle(ctx, token("g-other", "other@gmail.com"), web)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.LinkGoogle(ctx, u.ID, token("g-other", "other@gmail.com"), Reauth{Password: "correct-password"})
	wantCode(t, err, apperr.CodeConflict)

	m, err := f.svc.LinkGoogle(ctx, u.ID, token("g-a", "a@gmail.com"), Reauth{Password: "correct-password"})
	if err != nil {
		t.Fatal(err)
	}
	if m.Google == nil || m.Google.Provider != ProviderGoogle {
		t.Fatalf("methods = %+v", m)
	}
	// Signing in with it reaches this account.
	res, err := f.svc.SignInWithGoogle(ctx, token("g-a", "a@gmail.com"), web)
	if err != nil || res.User.ID != u.ID || res.User.ID == other.User.ID {
		t.Fatalf("res = %+v %v", res, err)
	}
	// Only one Google account at a time.
	_, err = f.svc.LinkGoogle(ctx, u.ID, token("g-b", "b@gmail.com"), Reauth{Password: "correct-password"})
	wantCode(t, err, apperr.CodeConflict)

	m, err = f.svc.UnlinkGoogle(ctx, u.ID)
	if err != nil || m.Google != nil {
		t.Fatalf("unlink = %+v %v", m, err)
	}
	_, err = f.svc.UnlinkGoogle(ctx, u.ID)
	wantCode(t, err, apperr.CodeNotFound)
	// That Google account no longer reaches this learner (its email differs from theirs, so a
	// new account is created for it).
	res, err = f.svc.SignInWithGoogle(ctx, token("g-a", "a@gmail.com"), web)
	if err != nil || res.User.ID == u.ID || !res.IsNewUser {
		t.Fatalf("res = %+v %v", res, err)
	}
}

func TestDeleteAccountTx(t *testing.T) {
	f := newFixture(t)
	u, current, other := f.passwordSession(t, "a@example.com", "correct-password")
	if err := f.svc.ForgotPassword(ctx, "a@example.com"); err != nil {
		t.Fatal(err)
	}
	resetLink := f.mail.last(t).Text

	if err := f.store.InTx(ctx, func(tx Tx) error { return f.svc.DeleteAccountTx(ctx, tx, u.ID) }); err != nil {
		t.Fatal(err)
	}
	if !f.revoked(t, current) || !f.revoked(t, other) {
		t.Fatal("every session ends")
	}
	_, err := f.svc.Me(ctx, u.ID)
	wantCode(t, err, apperr.CodeUnauthorized)
	_, err = f.svc.SignIn(ctx, "a@example.com", "correct-password", false, web)
	wantCode(t, err, apperr.CodeUnauthorized)
	// The reset link sent before the deletion is dead.
	token := resetLink[strings.Index(resetLink, "token=")+len("token="):]
	token = strings.Fields(token)[0]
	if err := f.svc.ResetPassword(ctx, token, "another-password"); err == nil {
		t.Fatal("reset link must not work after deletion")
	}
	// Deleting again: the account is gone.
	err = f.store.InTx(ctx, func(tx Tx) error { return f.svc.DeleteAccountTx(ctx, tx, u.ID) })
	wantCode(t, err, apperr.CodeUnauthorized)
	// The address is free: signing up creates a new account.
	f.advance(2 * time.Minute)
	res := f.signUp(t, "a@example.com")
	if !res.IsNewUser || res.User.ID == u.ID {
		t.Fatalf("res = %+v", res)
	}
}

func TestAccountChangesNeedActiveLearner(t *testing.T) {
	f := newFixture(t)
	u, sid := f.learnerSession(t, "a@example.com")
	if err := f.store.InTx(ctx, func(tx Tx) error { return tx.DeleteUser(ctx, u.ID, f.now) }); err != nil {
		t.Fatal(err)
	}
	_, err := f.svc.SignInMethods(ctx, u.ID)
	wantCode(t, err, apperr.CodeUnauthorized)
	_, err = f.svc.SendReauthCode(ctx, u.ID)
	wantCode(t, err, apperr.CodeUnauthorized)
	wantCode(t, f.svc.ChangePassword(ctx, u.ID, sid, Reauth{Code: "123456"}, "first-password"), apperr.CodeUnauthorized)
	_, err = f.svc.UnlinkGoogle(ctx, u.ID)
	wantCode(t, err, apperr.CodeUnauthorized)
}
