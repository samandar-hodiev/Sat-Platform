package profile

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/auth"
)

// My Profile (session 2.10, PRD §20 Account): editing the finished profile and deleting the
// account. Email, password and sign-in methods are the auth module's (auth/account.go).

// UpdateInput is the raw My Profile form. Nil fields are not changed; Phone "" removes it.
type UpdateInput struct {
	FirstName   *string
	LastName    *string
	Phone       *string
	DateOfBirth *string
	Timezone    *string
}

var errSetupFirst = apperr.Conflict("Finish setting up your profile first.").
	WithDetails(map[string]any{"reason": "profile_setup_required"})

// Update edits the learner's profile. Validation errors come back all at once, per field, with
// the same rules as Profile Setup.
func (s *Service) Update(ctx context.Context, userID string, in UpdateInput) (View, error) {
	acc, err := s.account(ctx, userID)
	if err != nil {
		return View{}, err
	}
	fields := map[string]string{}
	var up Update
	var msg string
	if in.FirstName != nil {
		var v string
		if v, msg = normalizeName(*in.FirstName); msg != "" {
			fields["first_name"] = msg
		}
		up.FirstName = &v
	}
	if in.LastName != nil {
		var v string
		if v, msg = normalizeName(*in.LastName); msg != "" {
			fields["last_name"] = msg
		}
		up.LastName = &v
	}
	if in.Phone != nil {
		var v string
		if v, msg = normalizePhone(*in.Phone); msg != "" {
			fields["phone"] = msg
		}
		up.Phone = &v
	}
	if in.DateOfBirth != nil {
		v, msg := parseDateOfBirth(*in.DateOfBirth, s.now().UTC())
		if msg != "" {
			fields["date_of_birth"] = msg
		}
		up.DateOfBirth = &v
	}
	if in.Timezone != nil {
		var v string
		if v, msg = normalizeTimezone(*in.Timezone); msg == "" && v == "" {
			msg = "is required"
		}
		if msg != "" {
			fields["timezone"] = msg
		}
		up.Timezone = &v
	}
	if len(fields) > 0 {
		return View{}, validation(fields)
	}

	p, err := s.store.Update(ctx, userID, up, s.now())
	switch {
	case errors.Is(err, ErrNotCompleted):
		return View{}, errSetupFirst
	case errors.Is(err, ErrNotFound):
		return View{}, errAccountGone
	case err != nil:
		return View{}, err
	}
	return view(acc, p), nil
}

// DeleteAccountInput confirms the deletion: the word DELETE typed by the learner, and their
// password (or an email code when the account has none).
type DeleteAccountInput struct {
	Confirmation string
	Reauth       auth.Reauth
}

// DeleteConfirmationWord must be typed to delete the account.
const DeleteConfirmationWord = "DELETE"

// DeleteAccount deletes the learner's account at once (decided for 2.10: soft delete, no grace
// period). In one transaction the auth module ends every session, removes the sign-in data and
// frees the email, and the profile row (name, phone, date of birth, avatar) is deleted. The
// avatar file is removed afterwards.
//
// Not yet covered, see docs/PROGRESS.md: cancelling a paid subscription (3.13, 5.01) and the
// audit log entry (1.02 helper).
func (s *Service) DeleteAccount(ctx context.Context, userID string, in DeleteAccountInput) error {
	if _, err := s.account(ctx, userID); err != nil {
		return err
	}
	if in.Confirmation != DeleteConfirmationWord {
		return validation(map[string]string{"confirmation": `type DELETE to confirm`})
	}
	if err := s.accounts.Reauthenticate(ctx, userID, in.Reauth); err != nil {
		return err
	}
	avatar, err := s.store.DeleteAccount(ctx, userID, func(tx pgx.Tx) error {
		return s.accounts.DeleteAccountTx(ctx, tx, userID)
	})
	if errors.Is(err, ErrNotFound) {
		return errAccountGone
	}
	if err != nil {
		return err
	}
	s.deleteFile(ctx, avatar)
	s.log.InfoContext(ctx, "learner account deleted", slog.String("user_id", userID))
	return nil
}
