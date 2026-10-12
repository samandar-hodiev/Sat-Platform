package auth

import (
	"context"
	"errors"
	"time"
)

// Store is the persistence the auth service needs. The PostgreSQL implementation
// (repository.go) is written once session 0.01 has created the users table; the auth tables
// (sessions, refresh tokens, verification codes, reset tokens, identities) are 1.01's own
// migrations. Tests use an in-memory implementation.
//
// Methods that must be atomic run inside InTx; the transaction holds row locks so two
// concurrent requests cannot both consume the same code or refresh token.
type Store interface {
	InTx(ctx context.Context, fn func(tx Tx) error) error
	Tx
}

// ErrNotFound is returned by lookups that find nothing.
var ErrNotFound = errors.New("not found")

// ErrPasswordAlreadySet: SetPasswordIfUnset found an existing password.
var ErrPasswordAlreadySet = errors.New("password already set")

type Tx interface {
	// Users.
	UserByID(ctx context.Context, id string) (User, error)
	UserByEmail(ctx context.Context, email string) (User, error)
	CreateUser(ctx context.Context, u NewUser) (User, error)
	SetPassword(ctx context.Context, userID, hash string) error
	// SetPasswordIfUnset sets the first password. ErrPasswordAlreadySet when one exists,
	// ErrNotFound when the user does not exist (or is deleted).
	SetPasswordIfUnset(ctx context.Context, userID, hash string) error
	MarkEmailVerified(ctx context.Context, userID string, at time.Time) error
	TouchLastLogin(ctx context.Context, userID string, at time.Time) error

	// External identities (Google). Keyed by provider + subject, never by email.
	UserIDByIdentity(ctx context.Context, provider, subject string) (string, error)
	LinkIdentity(ctx context.Context, userID, provider, subject, email string, at time.Time) error
	// IdentityProviders lists the linked external providers of a user (e.g. ["google"]).
	IdentityProviders(ctx context.Context, userID string) ([]string, error)

	// Email verification codes. LatestCode locks the newest unused code for the email.
	CodeStats(ctx context.Context, email string, since time.Time) (lastSentAt *time.Time, sent int, err error)
	CreateCode(ctx context.Context, c CodeRecord) error
	LatestCode(ctx context.Context, email string) (StoredCode, error)
	IncrementCodeAttempts(ctx context.Context, codeID string) error
	MarkCodeUsed(ctx context.Context, codeID string, at time.Time) error

	// Sessions and refresh tokens. Each sign-in is one session; every refresh rotates the token.
	CreateSession(ctx context.Context, s NewSession) (Session, error)
	SessionByID(ctx context.Context, id string) (Session, error)
	// RefreshTokenByHash locks the token row.
	RefreshTokenByHash(ctx context.Context, hash string) (RefreshToken, error)
	AddRefreshToken(ctx context.Context, sessionID, hash string, at time.Time) error
	MarkRefreshTokenUsed(ctx context.Context, hash string, at time.Time) error
	TouchSession(ctx context.Context, sessionID string, at time.Time) error
	RevokeSession(ctx context.Context, sessionID string, at time.Time) error
	RevokeUserSessions(ctx context.Context, userID string, at time.Time) error

	// Password reset tokens. CreateResetToken invalidates earlier unused tokens of the user.
	CreateResetToken(ctx context.Context, userID, hash string, expiresAt, at time.Time) error
	ResetTokenByHash(ctx context.Context, hash string) (ResetToken, error)
	MarkResetTokenUsed(ctx context.Context, hash string, at time.Time) error
}

// UserStatus values. Deleted or suspended accounts cannot sign in.
const (
	UserStatusActive    = "active"
	UserStatusSuspended = "suspended"
	UserStatusDeleted   = "deleted"
)

type User struct {
	ID              string
	Email           string
	Role            Role
	Status          string
	PasswordHash    string // empty for accounts without a password (e.g. Google only)
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
}

type NewUser struct {
	Email           string
	Role            Role
	EmailVerifiedAt *time.Time
}

type CodeRecord struct {
	Email     string
	Hash      string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type StoredCode struct {
	ID string
	CodeState
}

type NewSession struct {
	UserID    string
	ExpiresAt time.Time
	Remember  bool
	Client    ClientInfo
	CreatedAt time.Time
}

type Session struct {
	ID         string
	UserID     string
	ExpiresAt  time.Time
	Remember   bool
	RevokedAt  *time.Time
	LastUsedAt time.Time
}

type RefreshToken struct {
	SessionID string
	UsedAt    *time.Time
}

type ResetToken struct {
	UserID    string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// ClientInfo describes the device a session belongs to (shown later in "Active sessions").
type ClientInfo struct {
	Platform  string // web | android | ios
	UserAgent string
	IP        string
}
