package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/mailer"
)

// Account changes by the signed-in learner (My Profile, session 2.10; PRD §20 Account):
// changing the email, the password and the sign-in methods, and deleting the account.
//
// Sensitive changes ask the learner to confirm it is them (Reauth): the current password when
// the account has one, otherwise a code sent to the current email. Changing the email or the
// password keeps the current session and signs every other device out.

// Action code purposes (account_action_codes.purpose).
const (
	PurposeReauth      = "reauth"
	PurposeEmailChange = "email_change"
)

// Reauth is the learner's proof for a sensitive change. Password is required when the account
// has one; otherwise Code (from SendReauthCode).
type Reauth struct {
	Password string
	Code     string
}

type ActionCodeRecord struct {
	UserID    string
	Purpose   string
	Email     string
	Hash      string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type StoredActionCode struct {
	ID    string
	Email string
	CodeState
}

// Identity is a linked external sign-in.
type Identity struct {
	Provider string
	Email    string // reported by the provider when it was linked
	LinkedAt time.Time
}

// SignInMethods lists how the learner can sign in. Every active learner can also sign in with
// a code sent to their email.
type SignInMethods struct {
	Email         string
	EmailVerified bool
	HasPassword   bool
	Google        *Identity // nil: not linked
}

// ReauthMethod tells the client what Reauth needs: "password" or "email_code".
func (m SignInMethods) ReauthMethod() string {
	if m.HasPassword {
		return "password"
	}
	return "email_code"
}

var (
	errWrongPassword = fieldError("current_password", "This password is incorrect.")
	errCodeExpired   = fieldError("code", "This code has expired. Request a new one.")
	errCodeLocked    = fieldError("code", "Too many wrong attempts. Request a new code.")
)

// Per-user budget of password confirmations, so a stolen session cannot guess the password.
const (
	reauthPasswordAttempts = 5
	reauthPasswordWindow   = 15 * time.Minute
)

// activeLearner loads the signed-in learner inside tx.
func activeLearner(ctx context.Context, tx Tx, userID string) (User, error) {
	user, err := tx.UserByID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return User{}, errSessionEnded
	}
	if err != nil {
		return User{}, err
	}
	if checkCanSignIn(user) != nil {
		return User{}, errSessionEnded
	}
	return user, nil
}

// SignInMethods returns the learner's sign-in methods.
func (s *Service) SignInMethods(ctx context.Context, userID string) (SignInMethods, error) {
	user, err := s.Me(ctx, userID)
	if err != nil {
		return SignInMethods{}, err
	}
	m := SignInMethods{Email: user.Email, EmailVerified: user.EmailVerifiedAt != nil, HasPassword: user.PasswordHash != ""}
	id, err := s.store.Identity(ctx, userID, ProviderGoogle)
	switch {
	case err == nil:
		m.Google = &id
	case !errors.Is(err, ErrNotFound):
		return SignInMethods{}, err
	}
	return m, nil
}

// ---------------------------------------------------------------------------------------------
// Codes

// sendActionCode stores a new code for (user, purpose) and emails it to email.
func (s *Service) sendActionCode(ctx context.Context, userID, purpose, email string, write func(code string, ttl time.Duration) mailer.Message) (CodeSent, error) {
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
		last, sent, err := tx.ActionCodeStats(ctx, userID, purpose, now.Add(-24*time.Hour))
		if err != nil {
			return err
		}
		if wait, err := p.CheckCanSend(now, last, sent); err != nil {
			if errors.Is(err, ErrCodeResendTooSoon) {
				return apperr.New(apperr.CodeRateLimited, "Please wait before requesting another code.").
					WithDetails(map[string]any{"retry_after_seconds": int(wait.Seconds())})
			}
			return apperr.New(apperr.CodeRateLimited, "Too many codes requested today. Try again tomorrow.")
		}
		return tx.CreateActionCode(ctx, ActionCodeRecord{
			UserID: userID, Purpose: purpose, Email: email, Hash: s.actionCodeHash(purpose, email, code),
			ExpiresAt: now.Add(p.TTL), CreatedAt: now,
		})
	})
	if err != nil {
		return CodeSent{}, err
	}
	if err := s.mail.Send(ctx, write(code, p.TTL)); err != nil {
		return CodeSent{}, apperr.Wrap(apperr.CodeUnavailable, errEmailUnavailable.Message, err)
	}
	return CodeSent{ExpiresAt: now.Add(p.TTL), ResendAvailableAt: now.Add(p.ResendCooldown)}, nil
}

// actionCodeHash binds a code to its purpose as well as the email, so a reauth code can never
// confirm an email change (or a sign-up) and the other way round.
func (s *Service) actionCodeHash(purpose, email, code string) string {
	return s.codes.Hash(purpose+":"+email, code)
}

// consumeActionCode checks code against the newest code of (user, purpose) and marks it used.
// A wrong guess is counted and committed even though an error is returned. then (optional) runs
// in the same transaction; when it fails, the code stays unused.
func (s *Service) consumeActionCode(ctx context.Context, userID, purpose, code string, then func(tx Tx, stored StoredActionCode) error) error {
	now := s.now()
	p := s.cfg.Codes
	var mismatch bool
	var attempts int
	err := s.store.InTx(ctx, func(tx Tx) error {
		stored, err := tx.LatestActionCode(ctx, userID, purpose)
		if errors.Is(err, ErrNotFound) {
			return errCodeExpired
		}
		if err != nil {
			return err
		}
		// The purpose is bound into the hashed "email" (actionCodeHash).
		switch err := p.Verify(now, stored.CodeState, s.codes, purpose+":"+stored.Email, code); {
		case errors.Is(err, ErrCodeMismatch):
			mismatch, attempts = true, stored.Attempts+1
			return tx.IncrementActionCodeAttempts(ctx, stored.ID)
		case errors.Is(err, ErrCodeTooManyAttempts):
			return errCodeLocked
		case errors.Is(err, ErrCodeExpired), errors.Is(err, ErrCodeAlreadyUsed):
			return errCodeExpired
		case err != nil:
			return err
		}
		if err := tx.MarkActionCodeUsed(ctx, stored.ID, now); err != nil {
			return err
		}
		if then == nil {
			return nil
		}
		return then(tx, stored)
	})
	if mismatch {
		left := p.MaxAttempts - attempts
		if left <= 0 {
			return errCodeLocked
		}
		return fieldError("code", fmt.Sprintf("The code is incorrect. %d %s left.", left, plural(left, "attempt", "attempts")))
	}
	return err
}

// ---------------------------------------------------------------------------------------------
// Reauthentication

// SendReauthCode emails a confirmation code to the learner's current address. Only accounts
// without a password confirm with a code; the others use their password.
func (s *Service) SendReauthCode(ctx context.Context, userID string) (CodeSent, error) {
	user, err := s.Me(ctx, userID)
	if err != nil {
		return CodeSent{}, err
	}
	if user.PasswordHash != "" {
		return CodeSent{}, apperr.Conflict("Confirm with your password instead.")
	}
	return s.sendActionCode(ctx, userID, PurposeReauth, user.Email, func(code string, ttl time.Duration) mailer.Message {
		return mailer.Message{
			To:      user.Email,
			Subject: fmt.Sprintf("Your SAT Platform confirmation code: %s", code),
			Text: fmt.Sprintf("Use %s to confirm a change to your SAT Platform account.\n\nIt expires in %d minutes. If you didn't ask for it, someone may be using your account: sign out of other devices and contact support.",
				code, int(ttl.Minutes())),
		}
	})
}

// Reauthenticate confirms that the signed-in learner is the account owner (see Reauth). A code
// is spent even if the change that follows fails; the learner then requests a new one.
func (s *Service) Reauthenticate(ctx context.Context, userID string, r Reauth) error {
	user, err := s.Me(ctx, userID)
	if err != nil {
		return err
	}
	if user.PasswordHash != "" {
		if r.Password == "" {
			return fieldError("current_password", "Enter your current password.")
		}
		if len(r.Password) > 4*PasswordMaxLength {
			return errWrongPassword
		}
		if res := s.limiter.Allow(ctx, "account:reauth:"+userID, reauthPasswordAttempts, reauthPasswordWindow); !res.Allowed {
			return errTooManyAttempts
		}
		ok, err := VerifyPassword(r.Password, user.PasswordHash)
		if err != nil {
			return err
		}
		if !ok {
			return errWrongPassword
		}
		return nil
	}
	if r.Code == "" {
		return fieldError("code", "Enter the code we sent to your email.")
	}
	return s.consumeActionCode(ctx, userID, PurposeReauth, r.Code, nil)
}

// ---------------------------------------------------------------------------------------------
// Email

// EmailChangeStarted: a code was sent to the new address.
type EmailChangeStarted struct {
	Email string
	CodeSent
}

// StartEmailChange confirms the learner and emails a code to the new address. The email
// changes only when that code is confirmed (ConfirmEmailChange).
func (s *Service) StartEmailChange(ctx context.Context, userID, rawEmail string, r Reauth) (EmailChangeStarted, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return EmailChangeStarted{}, fieldError("new_email", "Enter a valid email address.")
	}
	user, err := s.Me(ctx, userID)
	if err != nil {
		return EmailChangeStarted{}, err
	}
	if email == user.Email {
		return EmailChangeStarted{}, fieldError("new_email", "This is already your email.")
	}
	if s.mail == nil {
		return EmailChangeStarted{}, errEmailUnavailable
	}
	// The learner is confirmed before anything reveals whether the address is registered.
	if err := s.Reauthenticate(ctx, userID, r); err != nil {
		return EmailChangeStarted{}, err
	}
	if err := s.checkEmailFree(ctx, userID, email); err != nil {
		return EmailChangeStarted{}, err
	}
	sent, err := s.sendActionCode(ctx, userID, PurposeEmailChange, email, func(code string, ttl time.Duration) mailer.Message {
		return mailer.Message{
			To:      email,
			Subject: fmt.Sprintf("Confirm your new SAT Platform email: %s", code),
			Text: fmt.Sprintf("Enter %s on SAT Platform to make this your account email.\n\nIt expires in %d minutes. If you didn't ask for it, you can ignore this email.",
				code, int(ttl.Minutes())),
		}
	})
	if err != nil {
		return EmailChangeStarted{}, err
	}
	return EmailChangeStarted{Email: email, CodeSent: sent}, nil
}

var errEmailTaken = fieldError("new_email", "This email is already used by another account.")

func (s *Service) checkEmailFree(ctx context.Context, userID, email string) error {
	other, err := s.store.UserByEmail(ctx, email)
	switch {
	case errors.Is(err, ErrNotFound):
		return nil
	case err != nil:
		return err
	case other.ID != userID:
		return errEmailTaken
	}
	return nil
}

// ConfirmEmailChange checks the code sent to the new address and makes it the account email.
// Other sessions are signed out; the previous address is told about the change.
func (s *Service) ConfirmEmailChange(ctx context.Context, userID, sessionID, code string) (SignInMethods, error) {
	var oldEmail, newEmail string
	err := s.consumeActionCode(ctx, userID, PurposeEmailChange, code, func(tx Tx, stored StoredActionCode) error {
		user, err := activeLearner(ctx, tx, userID)
		if err != nil {
			return err
		}
		oldEmail, newEmail = user.Email, stored.Email
		now := s.now()
		switch err := tx.ChangeEmail(ctx, userID, newEmail, now); {
		case errors.Is(err, ErrEmailTaken):
			return errEmailTaken
		case err != nil:
			return err
		}
		return tx.RevokeOtherSessions(ctx, userID, sessionID, now)
	})
	if err != nil {
		// The code is spent only when the change is committed: a taken address rolls it back.
		return SignInMethods{}, err
	}
	if s.mail != nil && oldEmail != newEmail {
		msg := mailer.Message{
			To:      oldEmail,
			Subject: "Your SAT Platform email was changed",
			Text: fmt.Sprintf("The email of your SAT Platform account was changed to %s. From now on, sign in with the new address.\n\nIf you didn't do this, contact support right away.",
				newEmail),
		}
		if err := s.mail.Send(ctx, msg); err != nil {
			s.log.ErrorContext(ctx, "send email change notice", slog.String("user_id", userID), slog.String("error", err.Error()))
		}
	}
	return s.SignInMethods(ctx, userID)
}

// ---------------------------------------------------------------------------------------------
// Password

// ChangePassword sets a new password: confirmed by the current one, or by an email code when
// the account has none yet (Google or email-code accounts). Other sessions are signed out.
func (s *Service) ChangePassword(ctx context.Context, userID, sessionID string, r Reauth, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return fieldError("new_password", passwordMessage(err))
	}
	if err := s.Reauthenticate(ctx, userID, r); err != nil {
		return err
	}
	if r.Password != "" && r.Password == newPassword {
		return fieldError("new_password", "Choose a password different from the current one.")
	}
	hash, err := HashPassword(newPassword, s.cfg.Argon2)
	if err != nil {
		return err
	}
	now := s.now()
	return s.store.InTx(ctx, func(tx Tx) error {
		if _, err := activeLearner(ctx, tx, userID); err != nil {
			return err
		}
		if err := tx.SetPassword(ctx, userID, hash); err != nil {
			return err
		}
		return tx.RevokeOtherSessions(ctx, userID, sessionID, now)
	})
}

// ---------------------------------------------------------------------------------------------
// Google

// LinkGoogle links the Google account of idToken. It needs Reauth: a linked Google account is a
// way in, so a borrowed session must not be able to add one.
func (s *Service) LinkGoogle(ctx context.Context, userID, idToken string, r Reauth) (SignInMethods, error) {
	if s.google == nil {
		return SignInMethods{}, errGoogleUnavailable
	}
	id, err := s.google.Verify(ctx, idToken)
	if errors.Is(err, ErrGoogleEmailNotVerified) {
		return SignInMethods{}, apperr.BadRequest("Your Google account email is not verified.")
	}
	if err != nil {
		s.log.InfoContext(ctx, "google link rejected", slog.String("reason", err.Error()))
		return SignInMethods{}, apperr.BadRequest("Google didn't confirm the account. Please try again.")
	}
	if err := s.Reauthenticate(ctx, userID, r); err != nil {
		return SignInMethods{}, err
	}
	now := s.now()
	err = s.store.InTx(ctx, func(tx Tx) error {
		if _, err := activeLearner(ctx, tx, userID); err != nil {
			return err
		}
		switch _, err := tx.Identity(ctx, userID, ProviderGoogle); {
		case err == nil:
			return apperr.Conflict("A Google account is already linked. Unlink it first.")
		case !errors.Is(err, ErrNotFound):
			return err
		}
		switch owner, err := tx.UserIDByIdentity(ctx, ProviderGoogle, id.Subject); {
		case err == nil && owner != userID:
			return apperr.Conflict("This Google account is linked to another SAT Platform account.")
		case err != nil && !errors.Is(err, ErrNotFound):
			return err
		}
		return tx.LinkIdentity(ctx, userID, ProviderGoogle, id.Subject, id.Email, now)
	})
	if err != nil {
		return SignInMethods{}, err
	}
	return s.SignInMethods(ctx, userID)
}

// UnlinkGoogle removes the linked Google account. The learner can always sign in with an email
// code, so this never locks them out.
func (s *Service) UnlinkGoogle(ctx context.Context, userID string) (SignInMethods, error) {
	err := s.store.InTx(ctx, func(tx Tx) error {
		if _, err := activeLearner(ctx, tx, userID); err != nil {
			return err
		}
		if err := tx.UnlinkIdentity(ctx, userID, ProviderGoogle); errors.Is(err, ErrNotFound) {
			return apperr.NotFound("No Google account is linked.")
		} else if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return SignInMethods{}, err
	}
	return s.SignInMethods(ctx, userID)
}

// ---------------------------------------------------------------------------------------------
// Account deletion

// DeleteAccountTx deletes the learner's sign-in data inside the caller's transaction (the
// profile module runs it together with deleting the profile, see profile.Service.DeleteAccount).
// The caller confirms the learner with Reauthenticate first.
func (s *Service) DeleteAccountTx(ctx context.Context, tx Tx, userID string) error {
	if _, err := activeLearner(ctx, tx, userID); err != nil {
		return err
	}
	return tx.DeleteUser(ctx, userID, s.now())
}
