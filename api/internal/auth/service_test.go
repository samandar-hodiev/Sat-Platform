package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/mailer"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
)

type outbox struct {
	mu   sync.Mutex
	msgs []mailer.Message
	fail error
}

func (o *outbox) Send(_ context.Context, m mailer.Message) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.fail != nil {
		return o.fail
	}
	o.msgs = append(o.msgs, m)
	return nil
}

func (o *outbox) last(t *testing.T) mailer.Message {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.msgs) == 0 {
		t.Fatal("no email sent")
	}
	return o.msgs[len(o.msgs)-1]
}

var codeRe = regexp.MustCompile(`\b(\d{6})\b`)

func (o *outbox) code(t *testing.T) string {
	t.Helper()
	m := codeRe.FindStringSubmatch(o.last(t).Text)
	if m == nil {
		t.Fatal("no code in email")
	}
	return m[1]
}

type fixture struct {
	svc   *Service
	store Store
	mail  *outbox
	now   time.Time
}

func (f *fixture) advance(d time.Duration) { f.now = f.now.Add(d) }

// newStore builds the Store for service tests: in memory by default; the PostgreSQL integration
// test (repository_integration_test.go) swaps it to run the same scenarios on a real database.
var newStore = func(t *testing.T) Store { return newMemStore() }

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{store: newStore(t), mail: &outbox{}, now: time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)}
	tokens, err := NewAccessTokens(testSecret, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	tokens.now = func() time.Time { return f.now }
	codes, _ := NewCodeHasher(testSecret)
	cfg := DefaultServiceConfig("http://localhost:3000")
	cfg.Argon2 = fastParams
	svc, err := NewService(ServiceDeps{
		Config: cfg, Store: f.store, Tokens: tokens, Codes: codes, Mail: f.mail,
		Limiter: ratelimit.NewMemory(), Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return f.now }
	f.svc = svc
	return f
}

var ctx = context.Background()
var web = ClientInfo{Platform: "web", IP: "127.0.0.1"}

func wantCode(t *testing.T, err error, code apperr.Code) *apperr.Error {
	t.Helper()
	var e *apperr.Error
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
	return e
}

func wantField(t *testing.T, err error, field, contains string) {
	t.Helper()
	e := wantCode(t, err, apperr.CodeValidation)
	fields, _ := e.Details["fields"].(map[string]string)
	if !strings.Contains(fields[field], contains) {
		t.Fatalf("field %s = %q, want it to contain %q", field, fields[field], contains)
	}
}

// signUp runs the whole email-code sign-up and returns the session.
func (f *fixture) signUp(t *testing.T, email string) SessionResult {
	t.Helper()
	if _, err := f.svc.StartSignUp(ctx, email); err != nil {
		t.Fatal(err)
	}
	res, err := f.svc.VerifySignUp(ctx, email, f.mail.code(t), web)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestSignUpCreatesVerifiedLearner(t *testing.T) {
	f := newFixture(t)
	sent, err := f.svc.StartSignUp(ctx, "  New@Example.com ")
	if err != nil {
		t.Fatal(err)
	}
	if !sent.ExpiresAt.Equal(f.now.Add(10*time.Minute)) || !sent.ResendAvailableAt.Equal(f.now.Add(time.Minute)) {
		t.Fatalf("sent = %+v", sent)
	}
	msg := f.mail.last(t)
	if msg.To != "new@example.com" {
		t.Fatalf("to = %s", msg.To)
	}

	res, err := f.svc.VerifySignUp(ctx, "new@example.com", f.mail.code(t), web)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsNewUser || res.User.Role != RoleLearner || res.User.EmailVerifiedAt == nil || res.User.PasswordHash != "" {
		t.Fatalf("user = %+v", res)
	}
	claims, err := f.svc.tokens.Verify(res.AccessToken)
	if err != nil || claims.UserID != res.User.ID {
		t.Fatalf("access token: %+v %v", claims, err)
	}
	if !res.RefreshTokenExpiresAt.Equal(f.now.Add(30 * 24 * time.Hour)) {
		t.Fatalf("refresh exp = %v", res.RefreshTokenExpiresAt)
	}
}

func TestSignUpCodeIsSingleUse(t *testing.T) {
	f := newFixture(t)
	_, _ = f.svc.StartSignUp(ctx, "a@example.com")
	code := f.mail.code(t)
	if _, err := f.svc.VerifySignUp(ctx, "a@example.com", code, web); err != nil {
		t.Fatal(err)
	}
	_, err := f.svc.VerifySignUp(ctx, "a@example.com", code, web)
	wantField(t, err, "code", "expired")
}

func TestSignUpExistingEmailSignsIn(t *testing.T) {
	f := newFixture(t)
	first := f.signUp(t, "a@example.com")
	f.advance(2 * time.Minute)
	second := f.signUp(t, "a@example.com")
	if second.IsNewUser || second.User.ID != first.User.ID {
		t.Fatalf("existing learner should sign in: %+v", second)
	}
}

func TestSignUpWrongCodeCountsAttempts(t *testing.T) {
	f := newFixture(t)
	_, _ = f.svc.StartSignUp(ctx, "a@example.com")
	good := f.mail.code(t)
	bad := "000000"
	if good == bad {
		bad = "111111"
	}
	for i := 4; i >= 1; i-- {
		_, err := f.svc.VerifySignUp(ctx, "a@example.com", bad, web)
		wantField(t, err, "code", "incorrect")
	}
	_, err := f.svc.VerifySignUp(ctx, "a@example.com", bad, web)
	wantField(t, err, "code", "Too many wrong attempts")
	// Even the right code is refused now.
	_, err = f.svc.VerifySignUp(ctx, "a@example.com", good, web)
	wantField(t, err, "code", "Too many wrong attempts")
}

func TestSignUpCodeExpires(t *testing.T) {
	f := newFixture(t)
	_, _ = f.svc.StartSignUp(ctx, "a@example.com")
	code := f.mail.code(t)
	f.advance(11 * time.Minute)
	_, err := f.svc.VerifySignUp(ctx, "a@example.com", code, web)
	wantField(t, err, "code", "expired")
}

func TestResendCooldownAndDailyCap(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.StartSignUp(ctx, "a@example.com"); err != nil {
		t.Fatal(err)
	}
	_, err := f.svc.StartSignUp(ctx, "a@example.com")
	e := wantCode(t, err, apperr.CodeRateLimited)
	if e.Details["retry_after_seconds"] != 60 {
		t.Fatalf("details = %v", e.Details)
	}
	for range 9 {
		f.advance(61 * time.Second)
		if _, err := f.svc.StartSignUp(ctx, "a@example.com"); err != nil {
			t.Fatal(err)
		}
	}
	f.advance(61 * time.Second)
	_, err = f.svc.StartSignUp(ctx, "a@example.com")
	wantCode(t, err, apperr.CodeRateLimited)
}

func TestStartSignUpValidatesAndNeedsMail(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.StartSignUp(ctx, "not-an-email")
	wantField(t, err, "email", "valid email")

	f.svc.mail = nil
	_, err = f.svc.StartSignUp(ctx, "a@example.com")
	wantCode(t, err, apperr.CodeUnavailable)

	f.svc.mail = &outbox{fail: errors.New("smtp down")}
	_, err = f.svc.StartSignUp(ctx, "b@example.com")
	wantCode(t, err, apperr.CodeUnavailable)
}

func (f *fixture) withPassword(t *testing.T, email, password string) User {
	t.Helper()
	res := f.signUp(t, email)
	hash, _ := HashPassword(password, fastParams)
	if err := f.store.SetPassword(ctx, res.User.ID, hash); err != nil {
		t.Fatal(err)
	}
	u, _ := f.store.UserByID(ctx, res.User.ID)
	return u
}

func TestSignIn(t *testing.T) {
	f := newFixture(t)
	u := f.withPassword(t, "a@example.com", "correct-password")

	res, err := f.svc.SignIn(ctx, "A@Example.com", "correct-password", false, web)
	if err != nil {
		t.Fatal(err)
	}
	if res.User.ID != u.ID || res.Remember || !res.RefreshTokenExpiresAt.Equal(f.now.Add(24*time.Hour)) {
		t.Fatalf("res = %+v", res)
	}

	_, err = f.svc.SignIn(ctx, "a@example.com", "wrong-password", true, web)
	e1 := wantCode(t, err, apperr.CodeUnauthorized)
	_, err = f.svc.SignIn(ctx, "nobody@example.com", "whatever", true, web)
	e2 := wantCode(t, err, apperr.CodeUnauthorized)
	if e1.Message != e2.Message {
		t.Fatal("unknown email and wrong password must look the same")
	}
}

func TestSignInWithoutPasswordLooksLikeWrongPassword(t *testing.T) {
	f := newFixture(t)
	f.signUp(t, "codeonly@example.com")
	_, err := f.svc.SignIn(ctx, "codeonly@example.com", "anything", true, web)
	wantCode(t, err, apperr.CodeUnauthorized)
}

func TestSignInRehashesOldParams(t *testing.T) {
	f := newFixture(t)
	res := f.signUp(t, "a@example.com")
	weak := fastParams
	weak.Iterations = 1
	weak.MemoryKiB = 32
	hash, _ := HashPassword("pw-12345678", weak)
	_ = f.store.SetPassword(ctx, res.User.ID, hash)

	if _, err := f.svc.SignIn(ctx, "a@example.com", "pw-12345678", true, web); err != nil {
		t.Fatal(err)
	}
	u, _ := f.store.UserByID(ctx, res.User.ID)
	if NeedsRehash(u.PasswordHash, fastParams) {
		t.Fatal("hash should be upgraded on sign-in")
	}
}

func TestSignInPerEmailLimit(t *testing.T) {
	f := newFixture(t)
	f.withPassword(t, "a@example.com", "correct-password")
	for range 10 {
		_, _ = f.svc.SignIn(ctx, "a@example.com", "wrong", true, web)
	}
	_, err := f.svc.SignIn(ctx, "a@example.com", "correct-password", true, web)
	wantCode(t, err, apperr.CodeRateLimited)
}

func TestSuspendedAndOwnerAccountsCannotSignIn(t *testing.T) {
	f := newFixture(t)
	u := f.withPassword(t, "a@example.com", "correct-password")
	mem := f.store.(*memStore)
	mem.d.users[u.ID] = func() User { x := mem.d.users[u.ID]; x.Status = UserStatusSuspended; return x }()
	_, err := f.svc.SignIn(ctx, "a@example.com", "correct-password", true, web)
	wantCode(t, err, apperr.CodeForbidden)

	mem.d.users[u.ID] = func() User { x := mem.d.users[u.ID]; x.Status = UserStatusActive; x.Role = "owner"; return x }()
	_, err = f.svc.SignIn(ctx, "a@example.com", "correct-password", true, web)
	wantCode(t, err, apperr.CodeForbidden)
}

func TestRefreshRotatesAndDetectsReuse(t *testing.T) {
	f := newFixture(t)
	first := f.signUp(t, "a@example.com")

	f.advance(time.Minute)
	second, err := f.svc.Refresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if second.RefreshToken == first.RefreshToken || second.AccessToken == "" {
		t.Fatal("refresh must rotate the token")
	}
	if !second.RefreshTokenExpiresAt.Equal(first.RefreshTokenExpiresAt) {
		t.Fatal("rotation must not extend the session's absolute lifetime")
	}

	// Another tab presents the old token a moment later: rejected, session kept.
	f.advance(5 * time.Second)
	_, err = f.svc.Refresh(ctx, first.RefreshToken)
	wantCode(t, err, apperr.CodeUnauthorized)
	third, err := f.svc.Refresh(ctx, second.RefreshToken)
	if err != nil {
		t.Fatalf("session must survive a concurrent refresh: %v", err)
	}

	// The old token shows up again much later: stolen. The whole session is revoked.
	f.advance(time.Hour)
	_, err = f.svc.Refresh(ctx, first.RefreshToken)
	wantCode(t, err, apperr.CodeUnauthorized)
	_, err = f.svc.Refresh(ctx, third.RefreshToken)
	wantCode(t, err, apperr.CodeUnauthorized)
}

func TestRefreshRejectsUnknownAndExpired(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Refresh(ctx, "")
	wantCode(t, err, apperr.CodeUnauthorized)
	_, err = f.svc.Refresh(ctx, "nope")
	wantCode(t, err, apperr.CodeUnauthorized)

	res := f.signUp(t, "a@example.com")
	f.advance(31 * 24 * time.Hour)
	_, err = f.svc.Refresh(ctx, res.RefreshToken)
	wantCode(t, err, apperr.CodeUnauthorized)
}

func TestLogoutRevokesOnlyThatSession(t *testing.T) {
	f := newFixture(t)
	f.withPassword(t, "a@example.com", "correct-password")
	laptop, _ := f.svc.SignIn(ctx, "a@example.com", "correct-password", true, web)
	phone, _ := f.svc.SignIn(ctx, "a@example.com", "correct-password", true, ClientInfo{Platform: "ios"})

	if err := f.svc.Logout(ctx, laptop.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Logout(ctx, laptop.RefreshToken); err != nil {
		t.Fatal("logout must be idempotent")
	}
	if err := f.svc.Logout(ctx, "unknown"); err != nil {
		t.Fatal(err)
	}
	_, err := f.svc.Refresh(ctx, laptop.RefreshToken)
	wantCode(t, err, apperr.CodeUnauthorized)
	if _, err := f.svc.Refresh(ctx, phone.RefreshToken); err != nil {
		t.Fatalf("other device must stay signed in: %v", err)
	}
}

func resetTokenFrom(t *testing.T, text string) string {
	t.Helper()
	m := regexp.MustCompile(`token=(\S+)`).FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("no reset link in %q", text)
	}
	tok, err := url.QueryUnescape(m[1])
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestPasswordResetFlow(t *testing.T) {
	f := newFixture(t)
	f.withPassword(t, "a@example.com", "old-password")
	session, _ := f.svc.SignIn(ctx, "a@example.com", "old-password", true, web)

	if err := f.svc.ForgotPassword(ctx, "A@example.com"); err != nil {
		t.Fatal(err)
	}
	msg := f.mail.last(t)
	if !strings.Contains(msg.Text, "http://localhost:3000/auth/reset-password?token=") {
		t.Fatalf("link missing: %s", msg.Text)
	}
	token := resetTokenFrom(t, msg.Text)

	err := f.svc.ResetPassword(ctx, token, "short")
	wantField(t, err, "password", "at least 8")

	if err := f.svc.ResetPassword(ctx, token, "new-password"); err != nil {
		t.Fatal(err)
	}
	// Single use.
	wantCode(t, f.svc.ResetPassword(ctx, token, "another-password"), apperr.CodeBadRequest)
	// Old password gone, new one works, other sessions are signed out.
	_, err = f.svc.SignIn(ctx, "a@example.com", "old-password", true, web)
	wantCode(t, err, apperr.CodeUnauthorized)
	if _, err := f.svc.SignIn(ctx, "a@example.com", "new-password", true, web); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Refresh(ctx, session.RefreshToken)
	wantCode(t, err, apperr.CodeUnauthorized)
}

func TestPasswordResetSetsFirstPasswordForCodeOnlyAccount(t *testing.T) {
	f := newFixture(t)
	f.signUp(t, "a@example.com")
	_ = f.svc.ForgotPassword(ctx, "a@example.com")
	if err := f.svc.ResetPassword(ctx, resetTokenFrom(t, f.mail.last(t).Text), "first-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.SignIn(ctx, "a@example.com", "first-password", true, web); err != nil {
		t.Fatal(err)
	}
}

func TestPasswordResetDoesNotRevealAccounts(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.ForgotPassword(ctx, "nobody@example.com"); err != nil {
		t.Fatal(err)
	}
	if len(f.mail.msgs) != 0 {
		t.Fatal("no email for unknown address")
	}
	f.withPassword(t, "a@example.com", "pw-12345678")
	sent := len(f.mail.msgs)
	for range 5 {
		if err := f.svc.ForgotPassword(ctx, "a@example.com"); err != nil {
			t.Fatal("per-email limit must be silent")
		}
	}
	if got := len(f.mail.msgs) - sent; got != 3 {
		t.Fatalf("emails sent = %d, want 3 (per-email limit)", got)
	}
}

func TestPasswordResetTokenExpiresAndIsSuperseded(t *testing.T) {
	f := newFixture(t)
	f.withPassword(t, "a@example.com", "pw-12345678")
	_ = f.svc.ForgotPassword(ctx, "a@example.com")
	first := resetTokenFrom(t, f.mail.last(t).Text)
	_ = f.svc.ForgotPassword(ctx, "a@example.com")
	second := resetTokenFrom(t, f.mail.last(t).Text)

	wantCode(t, f.svc.ResetPassword(ctx, first, "new-password"), apperr.CodeBadRequest)
	f.advance(61 * time.Minute)
	wantCode(t, f.svc.ResetPassword(ctx, second, "new-password"), apperr.CodeBadRequest)
	wantCode(t, f.svc.ResetPassword(ctx, "garbage", "new-password"), apperr.CodeBadRequest)
}

func TestMe(t *testing.T) {
	f := newFixture(t)
	res := f.signUp(t, "a@example.com")
	u, err := f.svc.Me(ctx, res.User.ID)
	if err != nil || u.Email != "a@example.com" {
		t.Fatalf("me = %+v %v", u, err)
	}
	_, err = f.svc.Me(ctx, "missing")
	wantCode(t, err, apperr.CodeUnauthorized)
}

func TestGoogleSignIn(t *testing.T) {
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	g := newFakeGoogle(t)
	f := newFixture(t)
	f.now = now
	f.svc.google = g.verifier(now)

	// Google disabled → unavailable.
	saved := f.svc.google
	f.svc.google = nil
	_, err := f.svc.SignInWithGoogle(ctx, "x", web)
	wantCode(t, err, apperr.CodeUnavailable)
	f.svc.google = saved

	// First sign-in creates a learner.
	res, err := f.svc.SignInWithGoogle(ctx, g.sign(g.claims(now), g.kid, g.key), web)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsNewUser || res.User.Email != "learner@gmail.com" {
		t.Fatalf("res = %+v", res)
	}
	// Second sign-in finds the same account by Google subject, even if the email changed.
	claims := g.claims(now)
	claims["email"] = "renamed@gmail.com"
	again, err := f.svc.SignInWithGoogle(ctx, g.sign(claims, g.kid, g.key), web)
	if err != nil || again.IsNewUser || again.User.ID != res.User.ID {
		t.Fatalf("again = %+v %v", again, err)
	}

	// An existing email account is linked on first Google use.
	existing := f.signUp(t, "teacher@school.uz")
	claims = g.claims(now)
	claims["sub"] = "other-subject"
	claims["email"] = "Teacher@School.uz"
	linked, err := f.svc.SignInWithGoogle(ctx, g.sign(claims, g.kid, g.key), web)
	if err != nil || linked.IsNewUser || linked.User.ID != existing.User.ID {
		t.Fatalf("linked = %+v %v", linked, err)
	}

	// Unverified Google email is refused.
	claims = g.claims(now)
	claims["sub"] = "third"
	claims["email_verified"] = false
	_, err = f.svc.SignInWithGoogle(ctx, g.sign(claims, g.kid, g.key), web)
	wantCode(t, err, apperr.CodeUnauthorized)
}

type prefillRecorder struct {
	calls map[string]GoogleProfile
	err   error
}

func (p *prefillRecorder) PrefillFromGoogle(_ context.Context, userID string, gp GoogleProfile) error {
	if p.calls == nil {
		p.calls = map[string]GoogleProfile{}
	}
	p.calls[userID] = gp
	return p.err
}

func TestGooglePrefillsNewLearnerProfileOnly(t *testing.T) {
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	g := newFakeGoogle(t)
	f := newFixture(t)
	f.now = now
	f.svc.google = g.verifier(now)
	rec := &prefillRecorder{}
	f.svc.prefill = rec

	res, err := f.svc.SignInWithGoogle(ctx, g.sign(g.claims(now), g.kid, g.key), web)
	if err != nil {
		t.Fatal(err)
	}
	if rec.calls[res.User.ID] != (GoogleProfile{FirstName: "Ali", LastName: "Valiyev"}) {
		t.Fatalf("prefill = %+v", rec.calls)
	}
	// A returning learner is not prefilled again.
	delete(rec.calls, res.User.ID)
	if _, err := f.svc.SignInWithGoogle(ctx, g.sign(g.claims(now), g.kid, g.key), web); err != nil {
		t.Fatal(err)
	}
	if len(rec.calls) != 0 {
		t.Fatal("prefill only on sign-up")
	}
	// A failing prefill never blocks sign-up.
	rec.err = errors.New("db down")
	claims := g.claims(now)
	claims["sub"], claims["email"] = "other", "other@gmail.com"
	if _, err := f.svc.SignInWithGoogle(ctx, g.sign(claims, g.kid, g.key), web); err != nil {
		t.Fatalf("prefill error must not fail sign-in: %v", err)
	}
}

func TestGoogleProfileNames(t *testing.T) {
	cases := []struct {
		in   GoogleIdentity
		want GoogleProfile
	}{
		{GoogleIdentity{GivenName: " Ali ", FamilyName: "Valiyev"}, GoogleProfile{"Ali", "Valiyev"}},
		{GoogleIdentity{Name: "Ali Vali o'g'li"}, GoogleProfile{"Ali", "Vali o'g'li"}},
		{GoogleIdentity{Name: "Madonna"}, GoogleProfile{"Madonna", ""}},
		{GoogleIdentity{}, GoogleProfile{}},
		{GoogleIdentity{GivenName: strings.Repeat("я", 120)}, GoogleProfile{strings.Repeat("я", 100), ""}},
	}
	for _, c := range cases {
		if got := googleProfile(c.in); got != c.want {
			t.Errorf("googleProfile(%+v) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestSetInitialPassword(t *testing.T) {
	f := newFixture(t)
	res := f.signUp(t, "a@example.com")

	wantField(t, f.svc.SetInitialPassword(ctx, res.User.ID, "short"), "password", "at least 8")
	if err := f.svc.SetInitialPassword(ctx, res.User.ID, "first-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.SignIn(ctx, "a@example.com", "first-password", true, web); err != nil {
		t.Fatalf("sign in with the new password: %v", err)
	}
	wantCode(t, f.svc.SetInitialPassword(ctx, res.User.ID, "another-password"), apperr.CodeConflict)
	wantCode(t, f.svc.SetInitialPassword(ctx, "00000000-0000-0000-0000-000000000000", "first-password"), apperr.CodeNotFound)
}

func TestAccountSignInMethods(t *testing.T) {
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	g := newFakeGoogle(t)
	f := newFixture(t)
	f.now = now
	f.svc.google = g.verifier(now)

	codeOnly := f.signUp(t, "code@example.com")
	acc, err := f.svc.Account(ctx, codeOnly.User.ID)
	if err != nil || acc.HasGoogle || acc.HasPassword() {
		t.Fatalf("code-only account = %+v %v", acc, err)
	}

	google, err := f.svc.SignInWithGoogle(ctx, g.sign(g.claims(now), g.kid, g.key), web)
	if err != nil {
		t.Fatal(err)
	}
	acc, err = f.svc.Account(ctx, google.User.ID)
	if err != nil || !acc.HasGoogle || acc.HasPassword() {
		t.Fatalf("google account = %+v %v", acc, err)
	}
	if err := f.svc.SetInitialPassword(ctx, google.User.ID, "first-password"); err != nil {
		t.Fatal(err)
	}
	acc, _ = f.svc.Account(ctx, google.User.ID)
	if !acc.HasGoogle || !acc.HasPassword() {
		t.Fatalf("google + password = %+v", acc)
	}
	_, err = f.svc.Account(ctx, "00000000-0000-0000-0000-000000000000")
	wantCode(t, err, apperr.CodeUnauthorized)
}
