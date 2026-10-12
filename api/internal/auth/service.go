package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/mailer"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
)

// Client-facing errors. Messages are shown to learners as is (web, Android, iOS).
var (
	errInvalidCredentials = apperr.Unauthorized("Email or password is incorrect.")
	errAccountUnavailable = apperr.Forbidden("This account can't be used to sign in here.")
	errSessionEnded       = apperr.Unauthorized("Your session has ended. Please sign in again.")
	errResetLinkInvalid   = apperr.BadRequest("This reset link is invalid or has expired.")
	errEmailUnavailable   = apperr.New(apperr.CodeUnavailable, "Email delivery is not available right now. Please try again later.")
	errGoogleUnavailable  = apperr.New(apperr.CodeUnavailable, "Google sign-in is not available right now.")
	errTooManyAttempts    = apperr.New(apperr.CodeRateLimited, "Too many attempts. Please wait a few minutes and try again.")
)

// Identity provider names stored with linked identities.
const ProviderGoogle = "google"

// ServiceConfig holds the lifetimes and limits the service enforces.
type ServiceConfig struct {
	RefreshTokenTTL      time.Duration // with "Remember me"
	ShortRefreshTokenTTL time.Duration // without
	ResetTokenTTL        time.Duration
	// RefreshReuseGrace: a rotated refresh token presented again within this window is treated
	// as a race between two tabs (rejected, session kept). Later reuse means the token was
	// stolen: the whole session is revoked.
	RefreshReuseGrace time.Duration
	// WebURL builds links in emails (password reset).
	WebURL string
	Argon2 Argon2Params
	Codes  CodePolicy
	// Per-email sign-in budget, on top of the per-IP limit of the route.
	SignInPerEmail       int
	SignInPerEmailWindow time.Duration
	// Password reset emails per address.
	ResetPerEmail       int
	ResetPerEmailWindow time.Duration
}

func DefaultServiceConfig(webURL string) ServiceConfig {
	return ServiceConfig{
		RefreshTokenTTL:      30 * 24 * time.Hour,
		ShortRefreshTokenTTL: 24 * time.Hour,
		ResetTokenTTL:        time.Hour,
		RefreshReuseGrace:    30 * time.Second,
		WebURL:               webURL,
		Argon2:               DefaultArgon2Params,
		Codes:                DefaultCodePolicy,
		SignInPerEmail:       10,
		SignInPerEmailWindow: 15 * time.Minute,
		ResetPerEmail:        3,
		ResetPerEmailWindow:  time.Hour,
	}
}

// GoogleProfile is what a new Google learner's profile can be prefilled with (PRD §7).
type GoogleProfile struct {
	FirstName string
	LastName  string
}

// ProfilePrefiller fills empty profile fields of a learner who just signed up with Google. The
// profile module (2.01) implements it and internal/server wires it in; auth never writes the
// profiles table itself. It runs after sign-up succeeded; an error is logged, not returned.
type ProfilePrefiller interface {
	PrefillFromGoogle(ctx context.Context, userID string, p GoogleProfile) error
}

type Service struct {
	cfg     ServiceConfig
	prefill ProfilePrefiller // nil: no prefill
	store   Store
	tokens  *AccessTokens
	codes   *CodeHasher
	google  *GoogleVerifier // nil: Google sign-in disabled
	mail    mailer.Mailer   // nil: email disabled
	limiter ratelimit.Limiter
	dummy   *DummyPasswordCheck
	log     *slog.Logger
	now     func() time.Time
}

type ServiceDeps struct {
	Config  ServiceConfig
	Store   Store
	Tokens  *AccessTokens
	Codes   *CodeHasher
	Google  *GoogleVerifier
	Mail    mailer.Mailer
	Limiter ratelimit.Limiter
	Log     *slog.Logger
	// Optional.
	ProfilePrefiller ProfilePrefiller
}

func NewService(d ServiceDeps) (*Service, error) {
	if d.Store == nil || d.Tokens == nil || d.Codes == nil || d.Limiter == nil || d.Log == nil {
		return nil, errors.New("auth: store, tokens, codes, limiter and log are required")
	}
	dummy, err := NewDummyPasswordCheck(d.Config.Argon2)
	if err != nil {
		return nil, err
	}
	return &Service{
		cfg: d.Config, store: d.Store, tokens: d.Tokens, codes: d.Codes, google: d.Google,
		mail: d.Mail, limiter: d.Limiter, dummy: dummy, log: d.Log, now: time.Now,
		prefill: d.ProfilePrefiller,
	}, nil
}

// TokenPair is what every successful sign-in or refresh returns.
type TokenPair struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

type SessionResult struct {
	TokenPair
	User      User
	IsNewUser bool
	// Remember tells the web session handler whether the refresh cookie should outlive the
	// browser session.
	Remember bool
}

// CodeSent tells the client when the code expires and when it may ask for another one.
type CodeSent struct {
	ExpiresAt         time.Time
	ResendAvailableAt time.Time
}

// ---------------------------------------------------------------------------------------------
// Sign-up and sign-in by email code

// StartSignUp emails a verification code. The answer is the same for new and existing
// addresses (an existing learner simply signs in with the code), so it reveals nothing.
func (s *Service) StartSignUp(ctx context.Context, rawEmail string) (CodeSent, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return CodeSent{}, fieldError("email", "Enter a valid email address.")
	}
	if s.mail == nil {
		return CodeSent{}, errEmailUnavailable
	}
	now := s.now()
	p := s.cfg.Codes

	code, err := NewCode()
	if err != nil {
		return CodeSent{}, err
	}
	err = s.store.InTx(ctx, func(tx Tx) error {
		last, sent, err := tx.CodeStats(ctx, email, now.Add(-24*time.Hour))
		if err != nil {
			return err
		}
		if wait, err := p.CheckCanSend(now, last, sent); err != nil {
			if errors.Is(err, ErrCodeResendTooSoon) {
				return apperr.New(apperr.CodeRateLimited, "Please wait before requesting another code.").
					WithDetails(map[string]any{"retry_after_seconds": int(wait.Seconds())})
			}
			return apperr.New(apperr.CodeRateLimited, "Too many codes requested for this email today. Try again tomorrow.")
		}
		return tx.CreateCode(ctx, CodeRecord{Email: email, Hash: s.codes.Hash(email, code), ExpiresAt: now.Add(p.TTL), CreatedAt: now})
	})
	if err != nil {
		return CodeSent{}, err
	}

	msg := mailer.Message{
		To:      email,
		Subject: fmt.Sprintf("Your SAT Platform code: %s", code),
		Text: fmt.Sprintf("Your verification code is %s.\n\nIt expires in %d minutes. If you didn't request it, you can ignore this email.",
			code, int(p.TTL.Minutes())),
	}
	if err := s.mail.Send(ctx, msg); err != nil {
		return CodeSent{}, apperr.Wrap(apperr.CodeUnavailable, errEmailUnavailable.Message, err)
	}
	return CodeSent{ExpiresAt: now.Add(p.TTL), ResendAvailableAt: now.Add(p.ResendCooldown)}, nil
}

// VerifySignUp checks the code. A new address gets a learner account; an existing learner is
// signed in (the code proves they own the mailbox).
func (s *Service) VerifySignUp(ctx context.Context, rawEmail, code string, client ClientInfo) (SessionResult, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return SessionResult{}, fieldError("email", "Enter a valid email address.")
	}
	now := s.now()
	p := s.cfg.Codes

	var user User
	var isNew bool
	var pair TokenPair
	var mismatch bool
	err = s.store.InTx(ctx, func(tx Tx) error {
		stored, err := tx.LatestCode(ctx, email)
		if errors.Is(err, ErrNotFound) {
			return fieldError("code", "This code has expired. Request a new one.")
		}
		if err != nil {
			return err
		}
		switch err := p.Verify(now, stored.CodeState, s.codes, email, code); {
		case errors.Is(err, ErrCodeMismatch):
			// Count the wrong guess and commit it; the error is returned after the transaction.
			mismatch = true
			return tx.IncrementCodeAttempts(ctx, stored.ID)
		case errors.Is(err, ErrCodeTooManyAttempts):
			return fieldError("code", "Too many wrong attempts. Request a new code.")
		case errors.Is(err, ErrCodeExpired), errors.Is(err, ErrCodeAlreadyUsed):
			return fieldError("code", "This code has expired. Request a new one.")
		case err != nil:
			return err
		}
		if err := tx.MarkCodeUsed(ctx, stored.ID, now); err != nil {
			return err
		}

		user, err = tx.UserByEmail(ctx, email)
		switch {
		case errors.Is(err, ErrNotFound):
			user, err = tx.CreateUser(ctx, NewUser{Email: email, Role: RoleLearner, EmailVerifiedAt: &now})
			if err != nil {
				return err
			}
			isNew = true
		case err != nil:
			return err
		default:
			if err := checkCanSignIn(user); err != nil {
				return err
			}
			if user.EmailVerifiedAt == nil {
				if err := tx.MarkEmailVerified(ctx, user.ID, now); err != nil {
					return err
				}
			}
		}
		pair, err = s.startSession(ctx, tx, user, true, client, now)
		return err
	})
	if mismatch {
		left := p.MaxAttempts - 1
		if stored, err := s.store.LatestCode(ctx, email); err == nil {
			left = p.MaxAttempts - stored.Attempts
		}
		if left <= 0 {
			return SessionResult{}, fieldError("code", "Too many wrong attempts. Request a new code.")
		}
		return SessionResult{}, fieldError("code", fmt.Sprintf("The code is incorrect. %d %s left.", left, plural(left, "attempt", "attempts")))
	}
	if err != nil {
		return SessionResult{}, err
	}
	return SessionResult{TokenPair: pair, User: user, IsNewUser: isNew, Remember: true}, nil
}

// ---------------------------------------------------------------------------------------------
// Password sign-in

func (s *Service) SignIn(ctx context.Context, rawEmail, password string, remember bool, client ClientInfo) (SessionResult, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil || password == "" || len(password) > 4*PasswordMaxLength {
		return SessionResult{}, errInvalidCredentials
	}
	if res := s.limiter.Allow(ctx, "auth:sign-in:email:"+email, s.cfg.SignInPerEmail, s.cfg.SignInPerEmailWindow); !res.Allowed {
		return SessionResult{}, errTooManyAttempts
	}

	user, err := s.store.UserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) || (err == nil && user.PasswordHash == "") {
		s.dummy.Run(password) // same timing as a real check: no account enumeration
		return SessionResult{}, errInvalidCredentials
	}
	if err != nil {
		return SessionResult{}, err
	}
	ok, err := VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return SessionResult{}, fmt.Errorf("verify password for user %s: %w", user.ID, err)
	}
	if !ok {
		return SessionResult{}, errInvalidCredentials
	}
	if err := checkCanSignIn(user); err != nil {
		return SessionResult{}, err
	}

	now := s.now()
	var pair TokenPair
	err = s.store.InTx(ctx, func(tx Tx) error {
		if NeedsRehash(user.PasswordHash, s.cfg.Argon2) {
			if hash, err := HashPassword(password, s.cfg.Argon2); err == nil {
				if err := tx.SetPassword(ctx, user.ID, hash); err != nil {
					return err
				}
			}
		}
		var err error
		pair, err = s.startSession(ctx, tx, user, remember, client, now)
		return err
	})
	if err != nil {
		return SessionResult{}, err
	}
	return SessionResult{TokenPair: pair, User: user, Remember: remember}, nil
}

// ---------------------------------------------------------------------------------------------
// Google

// SignInWithGoogle signs in by Google ID token. Accounts are matched by the Google subject;
// on first use an existing account with the same verified email is linked, otherwise a new
// learner account is created.
func (s *Service) SignInWithGoogle(ctx context.Context, idToken string, client ClientInfo) (SessionResult, error) {
	if s.google == nil {
		return SessionResult{}, errGoogleUnavailable
	}
	id, err := s.google.Verify(ctx, idToken)
	if errors.Is(err, ErrGoogleEmailNotVerified) {
		return SessionResult{}, apperr.Unauthorized("Your Google account email is not verified.")
	}
	if err != nil {
		s.log.InfoContext(ctx, "google sign-in rejected", slog.String("reason", err.Error()))
		return SessionResult{}, apperr.Unauthorized("Google sign-in failed. Please try again.")
	}

	now := s.now()
	var user User
	var isNew bool
	var pair TokenPair
	err = s.store.InTx(ctx, func(tx Tx) error {
		userID, err := tx.UserIDByIdentity(ctx, ProviderGoogle, id.Subject)
		switch {
		case err == nil:
			user, err = tx.UserByID(ctx, userID)
			if err != nil {
				return err
			}
		case errors.Is(err, ErrNotFound):
			user, err = tx.UserByEmail(ctx, id.Email)
			switch {
			case errors.Is(err, ErrNotFound):
				user, err = tx.CreateUser(ctx, NewUser{Email: id.Email, Role: RoleLearner, EmailVerifiedAt: &now})
				if err != nil {
					return err
				}
				isNew = true
			case err != nil:
				return err
			}
			if err := checkCanSignIn(user); err != nil {
				return err
			}
			if err := tx.LinkIdentity(ctx, user.ID, ProviderGoogle, id.Subject, id.Email, now); err != nil {
				return err
			}
			if user.EmailVerifiedAt == nil {
				if err := tx.MarkEmailVerified(ctx, user.ID, now); err != nil {
					return err
				}
			}
		default:
			return err
		}
		if err := checkCanSignIn(user); err != nil {
			return err
		}
		pair, err = s.startSession(ctx, tx, user, true, client, now)
		return err
	})
	if err != nil {
		return SessionResult{}, err
	}
	if isNew && s.prefill != nil {
		if p := googleProfile(id); p != (GoogleProfile{}) {
			if err := s.prefill.PrefillFromGoogle(ctx, user.ID, p); err != nil {
				s.log.ErrorContext(ctx, "prefill profile from Google", slog.String("user_id", user.ID), slog.String("error", err.Error()))
			}
		}
	}
	return SessionResult{TokenPair: pair, User: user, IsNewUser: isNew, Remember: true}, nil
}

// googleProfile prefers given/family name and falls back to splitting the full name at the
// first space. Values are trimmed and capped at 100 characters (profiles limit).
func googleProfile(id GoogleIdentity) GoogleProfile {
	first, last := strings.TrimSpace(id.GivenName), strings.TrimSpace(id.FamilyName)
	if first == "" && last == "" {
		name := strings.TrimSpace(id.Name)
		first, last, _ = strings.Cut(name, " ")
		first, last = strings.TrimSpace(first), strings.TrimSpace(last)
	}
	return GoogleProfile{FirstName: capRunes(first, 100), LastName: capRunes(last, 100)}
}

func capRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// ---------------------------------------------------------------------------------------------
// First password (Profile Setup, 2.01)

// SetInitialPassword sets the first password of an account that has none (email-code or Google
// sign-up). It fails with CONFLICT when a password exists: changing one goes through the
// current password (2.10) or a reset link.
func (s *Service) SetInitialPassword(ctx context.Context, userID, password string) error {
	return s.store.InTx(ctx, func(tx Tx) error { return s.SetInitialPasswordTx(ctx, tx, userID, password) })
}

// SetInitialPasswordTx is SetInitialPassword inside the caller's transaction, so a module can
// save its own rows and the password atomically:
//
//	pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
//		// … upsert the profile …
//		return authService.SetInitialPasswordTx(ctx, auth.PgxTx(tx), userID, password)
//	})
func (s *Service) SetInitialPasswordTx(ctx context.Context, tx Tx, userID, password string) error {
	if err := ValidatePassword(password); err != nil {
		return fieldError("password", passwordMessage(err))
	}
	hash, err := HashPassword(password, s.cfg.Argon2)
	if err != nil {
		return err
	}
	switch err := tx.SetPasswordIfUnset(ctx, userID, hash); {
	case errors.Is(err, ErrPasswordAlreadySet):
		return apperr.Conflict("A password is already set for this account.")
	case errors.Is(err, ErrNotFound):
		return apperr.NotFound("Account not found.")
	default:
		return err
	}
}

// ---------------------------------------------------------------------------------------------
// Sessions

// Refresh rotates the refresh token and returns a new pair.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (SessionResult, error) {
	if refreshToken == "" {
		return SessionResult{}, errSessionEnded
	}
	hash := HashOpaqueToken(refreshToken)
	now := s.now()

	var result SessionResult
	var stolen *Session
	err := s.store.InTx(ctx, func(tx Tx) error {
		rt, err := tx.RefreshTokenByHash(ctx, hash)
		if errors.Is(err, ErrNotFound) {
			return errSessionEnded
		}
		if err != nil {
			return err
		}
		sess, err := tx.SessionByID(ctx, rt.SessionID)
		if err != nil {
			return err
		}
		if sess.RevokedAt != nil || !now.Before(sess.ExpiresAt) {
			return errSessionEnded
		}
		if rt.UsedAt != nil {
			if now.Sub(*rt.UsedAt) <= s.cfg.RefreshReuseGrace {
				return errSessionEnded // concurrent refresh from another tab; keep the session
			}
			stolen = &sess
			return tx.RevokeSession(ctx, sess.ID, now)
		}
		user, err := tx.UserByID(ctx, sess.UserID)
		if err != nil {
			return err
		}
		if err := checkCanSignIn(user); err != nil {
			return tx.RevokeSession(ctx, sess.ID, now)
		}

		newToken, newHash, err := NewOpaqueToken()
		if err != nil {
			return err
		}
		if err := tx.MarkRefreshTokenUsed(ctx, hash, now); err != nil {
			return err
		}
		if err := tx.AddRefreshToken(ctx, sess.ID, newHash, now); err != nil {
			return err
		}
		if err := tx.TouchSession(ctx, sess.ID, now); err != nil {
			return err
		}
		access, accessExp, err := s.tokens.Issue(user.ID, user.Role, sess.ID)
		if err != nil {
			return err
		}
		result = SessionResult{
			TokenPair: TokenPair{AccessToken: access, AccessTokenExpiresAt: accessExp,
				RefreshToken: newToken, RefreshTokenExpiresAt: sess.ExpiresAt},
			User: user, Remember: sess.Remember,
		}
		return nil
	})
	if stolen != nil {
		// Audit logging (1.02 helper) and a security notification (3.14) hook in here.
		s.log.WarnContext(ctx, "refresh token reuse detected; session revoked",
			slog.String("session_id", stolen.ID), slog.String("user_id", stolen.UserID))
		return SessionResult{}, errSessionEnded
	}
	if err != nil {
		return SessionResult{}, err
	}
	return result, nil
}

// Logout revokes the session of the refresh token. Unknown tokens are ignored, so logout is
// idempotent and reveals nothing.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	now := s.now()
	return s.store.InTx(ctx, func(tx Tx) error {
		rt, err := tx.RefreshTokenByHash(ctx, HashOpaqueToken(refreshToken))
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return tx.RevokeSession(ctx, rt.SessionID, now)
	})
}

// Account is a user with the sign-in methods they can use.
type Account struct {
	User
	// HasGoogle: a Google identity is linked. HasPassword is User.PasswordHash != "".
	// Every active learner can also sign in with an email code.
	HasGoogle bool
}

func (a Account) HasPassword() bool { return a.PasswordHash != "" }

// Account returns the user and their sign-in methods (Profile Setup 2.01, My Profile 2.10).
func (s *Service) Account(ctx context.Context, userID string) (Account, error) {
	user, err := s.Me(ctx, userID)
	if err != nil {
		return Account{}, err
	}
	providers, err := s.store.IdentityProviders(ctx, userID)
	if err != nil {
		return Account{}, err
	}
	return Account{User: user, HasGoogle: slices.Contains(providers, ProviderGoogle)}, nil
}

// Me returns the signed-in user.
func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	user, err := s.store.UserByID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return User{}, errSessionEnded
	}
	if err != nil {
		return User{}, err
	}
	if err := checkCanSignIn(user); err != nil {
		return User{}, errSessionEnded
	}
	return user, nil
}

// ---------------------------------------------------------------------------------------------
// Password reset

// ForgotPassword emails a reset link when the address belongs to an active learner. It returns
// nil in every other case too, so the response never reveals which emails are registered.
func (s *Service) ForgotPassword(ctx context.Context, rawEmail string) error {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return fieldError("email", "Enter a valid email address.")
	}
	if s.mail == nil {
		return errEmailUnavailable
	}
	if res := s.limiter.Allow(ctx, "auth:reset:email:"+email, s.cfg.ResetPerEmail, s.cfg.ResetPerEmailWindow); !res.Allowed {
		return nil // silently: a visible error would confirm the address exists
	}
	user, err := s.store.UserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if checkCanSignIn(user) != nil {
		return nil
	}

	token, hash, err := NewOpaqueToken()
	if err != nil {
		return err
	}
	now := s.now()
	// Supersede earlier links and store the new one atomically.
	if err := s.store.InTx(ctx, func(tx Tx) error {
		return tx.CreateResetToken(ctx, user.ID, hash, now.Add(s.cfg.ResetTokenTTL), now)
	}); err != nil {
		return err
	}
	link := s.cfg.WebURL + "/auth/reset-password?token=" + url.QueryEscape(token)
	msg := mailer.Message{
		To:      email,
		Subject: "Reset your SAT Platform password",
		Text: fmt.Sprintf("Use this link to set a new password:\n\n%s\n\nIt expires in %d minutes and works once. If you didn't ask for it, ignore this email; your password stays the same.",
			link, int(s.cfg.ResetTokenTTL.Minutes())),
	}
	if err := s.mail.Send(ctx, msg); err != nil {
		// Logged, not returned: the answer must not differ for existing accounts.
		s.log.ErrorContext(ctx, "send password reset email", slog.String("error", err.Error()))
	}
	return nil
}

// ResetPassword sets a new password from a reset link and signs the user out everywhere.
func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if err := ValidatePassword(password); err != nil {
		return fieldError("password", passwordMessage(err))
	}
	if token == "" {
		return errResetLinkInvalid
	}
	hash := HashOpaqueToken(token)
	newHash, err := HashPassword(password, s.cfg.Argon2)
	if err != nil {
		return err
	}
	now := s.now()
	return s.store.InTx(ctx, func(tx Tx) error {
		rt, err := tx.ResetTokenByHash(ctx, hash)
		if errors.Is(err, ErrNotFound) {
			return errResetLinkInvalid
		}
		if err != nil {
			return err
		}
		if rt.UsedAt != nil || !now.Before(rt.ExpiresAt) {
			return errResetLinkInvalid
		}
		user, err := tx.UserByID(ctx, rt.UserID)
		if err != nil {
			return err
		}
		if checkCanSignIn(user) != nil {
			return errResetLinkInvalid
		}
		if err := tx.MarkResetTokenUsed(ctx, hash, now); err != nil {
			return err
		}
		if err := tx.SetPassword(ctx, user.ID, newHash); err != nil {
			return err
		}
		if user.EmailVerifiedAt == nil {
			// The link arrived by email, so the address is proven.
			if err := tx.MarkEmailVerified(ctx, user.ID, now); err != nil {
				return err
			}
		}
		return tx.RevokeUserSessions(ctx, user.ID, now)
	})
}

// ---------------------------------------------------------------------------------------------

func (s *Service) startSession(ctx context.Context, tx Tx, user User, remember bool, client ClientInfo, now time.Time) (TokenPair, error) {
	ttl := s.cfg.ShortRefreshTokenTTL
	if remember {
		ttl = s.cfg.RefreshTokenTTL
	}
	sess, err := tx.CreateSession(ctx, NewSession{UserID: user.ID, ExpiresAt: now.Add(ttl), Remember: remember, Client: client, CreatedAt: now})
	if err != nil {
		return TokenPair{}, err
	}
	refresh, refreshHash, err := NewOpaqueToken()
	if err != nil {
		return TokenPair{}, err
	}
	if err := tx.AddRefreshToken(ctx, sess.ID, refreshHash, now); err != nil {
		return TokenPair{}, err
	}
	if err := tx.TouchLastLogin(ctx, user.ID, now); err != nil {
		return TokenPair{}, err
	}
	access, accessExp, err := s.tokens.Issue(user.ID, user.Role, sess.ID)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, AccessTokenExpiresAt: accessExp, RefreshToken: refresh, RefreshTokenExpiresAt: sess.ExpiresAt}, nil
}

// checkCanSignIn: only active learner accounts use these endpoints. Owners sign in at
// /owner/login (1.02), with MFA.
func checkCanSignIn(u User) error {
	if u.Status != UserStatusActive || u.Role != RoleLearner {
		return errAccountUnavailable
	}
	return nil
}

func fieldError(field, message string) error {
	return apperr.New(apperr.CodeValidation, "Some fields are invalid").
		WithDetails(map[string]any{"fields": map[string]string{field: message}})
}

func passwordMessage(err error) string {
	if errors.Is(err, ErrPasswordTooLong) {
		return fmt.Sprintf("Use at most %d characters.", PasswordMaxLength)
	}
	return fmt.Sprintf("Use at least %d characters.", PasswordMinLength)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
