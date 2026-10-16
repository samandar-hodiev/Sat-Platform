package profile

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/samandar-hodiev/sat-platform/api/internal/auth"
)

var (
	ErrNotFound = errors.New("profile: not found")
	// ErrAlreadyCompleted: Profile Setup was finished before; later edits belong to My Profile (2.10).
	ErrAlreadyCompleted = errors.New("profile: setup already completed")
	// ErrNotCompleted: My Profile edits a finished profile only; Profile Setup comes first.
	ErrNotCompleted = errors.New("profile: setup not completed")
)

// Profile is one profiles row. Empty strings and nil pointers are NULL columns.
type Profile struct {
	UserID      string
	FirstName   string
	LastName    string
	Phone       string
	DateOfBirth *time.Time
	AvatarKey   string
	Timezone    string
	CompletedAt *time.Time
	UpdatedAt   time.Time
}

// Setup is the validated Profile Setup form.
type Setup struct {
	FirstName   string
	LastName    string
	Phone       string // E.164 or empty
	DateOfBirth time.Time
	Timezone    string // empty keeps the current value
}

// Update is a validated My Profile edit (2.10). Nil fields stay as they are; Phone "" clears it.
type Update struct {
	FirstName   *string
	LastName    *string
	Phone       *string
	DateOfBirth *time.Time
	Timezone    *string
}

// Store persists profiles (PostgreSQL in production, memory in unit tests).
type Store interface {
	// Get returns the profile, or ErrNotFound when the user has no row yet.
	Get(ctx context.Context, userID string) (Profile, error)
	// CompleteSetup writes the setup fields and marks the profile complete, creating the row if
	// needed, then runs also (when not nil) in the same transaction. ErrAlreadyCompleted when the
	// profile was already complete; nothing is written when either step fails.
	CompleteSetup(ctx context.Context, userID string, in Setup, now time.Time, also func(pgx.Tx) error) (Profile, error)
	// Prefill sets first and last name where they are still empty, on a profile that is not
	// complete (Google sign-up, PRD §7). The learner confirms them in Profile Setup.
	Prefill(ctx context.Context, userID, firstName, lastName string) error
	// SetAvatar stores the new avatar key and returns the previous one ("" if none).
	SetAvatar(ctx context.Context, userID, key string) (previous string, err error)
	// ClearAvatar removes the avatar key and returns the previous one ("" if none).
	ClearAvatar(ctx context.Context, userID string) (previous string, err error)
	// Update edits a completed profile (My Profile, 2.10). ErrNotCompleted when Profile Setup
	// is not finished (or there is no profile row).
	Update(ctx context.Context, userID string, in Update, now time.Time) (Profile, error)
	// DeleteAccount runs also (the auth module's account deletion) and deletes the profile row
	// in one transaction, returning the avatar key to delete from storage ("" if none). With no
	// profile row only also runs.
	DeleteAccount(ctx context.Context, userID string, also func(pgx.Tx) error) (avatarKey string, err error)
}

// Account is what Profile Setup needs from the auth module.
type Account struct {
	Email       string
	HasPassword bool
	// HasGoogle: a Google identity is linked, so a password is optional (decided for 2.01).
	HasGoogle bool
}

// Accounts is the auth module's side of Profile Setup and My Profile (an adapter over auth.Service, wired in
// internal/server).
type Accounts interface {
	// Account returns the signed-in learner's account. ErrNotFound when it no longer exists.
	Account(ctx context.Context, userID string) (Account, error)
	// SetInitialPasswordTx sets the first password inside the profile transaction, so the
	// profile and the password are saved together or not at all. It returns API errors
	// (VALIDATION_ERROR on "password", CONFLICT when a password exists) as they are.
	SetInitialPasswordTx(ctx context.Context, tx pgx.Tx, userID, password string) error
	// Reauthenticate confirms the learner before account deletion (password, or an email code
	// when the account has none). API errors pass through.
	Reauthenticate(ctx context.Context, userID string, r auth.Reauth) error
	// DeleteAccountTx deletes the sign-in data inside the profile transaction.
	DeleteAccountTx(ctx context.Context, tx pgx.Tx, userID string) error
}
