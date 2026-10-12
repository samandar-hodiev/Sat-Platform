package auth

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is the production Store. Tables: users (0.01) and the auth tables of 1.01
// (user_identities, email_verification_codes, auth_sessions, auth_refresh_tokens,
// password_reset_tokens). Deleted accounts (users.deleted_at) are invisible to auth.
type PostgresStore struct {
	pool *pgxpool.Pool
	queries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, queries: queries{q: pool}}
}

// InTx runs fn in one transaction; any error rolls everything back.
func (s *PostgresStore) InTx(ctx context.Context, fn func(tx Tx) error) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(queries{q: tx})
	})
}

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type queries struct{ q querier }

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	// A malformed id (not a UUID) cannot match any row.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return ErrNotFound
	}
	return err
}

// mustAffect turns "no row updated" into ErrNotFound.
func mustAffect(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const userColumns = `id, email, role, status, COALESCE(password_hash, ''), email_verified_at, created_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	var role string
	err := row.Scan(&u.ID, &u.Email, &role, &u.Status, &u.PasswordHash, &u.EmailVerifiedAt, &u.CreatedAt)
	u.Role = Role(role)
	return u, notFound(err)
}

func (r queries) UserByID(ctx context.Context, id string) (User, error) {
	return scanUser(r.q.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r queries) UserByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(r.q.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1 AND deleted_at IS NULL`, email))
}

func (r queries) CreateUser(ctx context.Context, n NewUser) (User, error) {
	return scanUser(r.q.QueryRow(ctx,
		`INSERT INTO users (email, role, email_verified_at) VALUES ($1, $2, $3) RETURNING `+userColumns,
		n.Email, string(n.Role), n.EmailVerifiedAt))
}

func (r queries) SetPassword(ctx context.Context, userID, hash string) error {
	return mustAffect(r.q.Exec(ctx,
		`UPDATE users SET password_hash = $2 WHERE id = $1 AND deleted_at IS NULL`, userID, hash))
}

func (r queries) SetPasswordIfUnset(ctx context.Context, userID, hash string) error {
	var set bool
	err := r.q.QueryRow(ctx,
		`SELECT password_hash IS NOT NULL FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&set)
	if err != nil {
		return notFound(err)
	}
	if set {
		return ErrPasswordAlreadySet
	}
	_, err = r.q.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, userID, hash)
	return err
}

// PgxTx adapts a pgx transaction owned by another module to the auth Tx, so auth writes can
// join it (e.g. Profile Setup saves the profile and the first password atomically).
func PgxTx(tx pgx.Tx) Tx { return queries{q: tx} }

func (r queries) MarkEmailVerified(ctx context.Context, userID string, at time.Time) error {
	_, err := r.q.Exec(ctx,
		`UPDATE users SET email_verified_at = $2 WHERE id = $1 AND email_verified_at IS NULL`, userID, at)
	return err
}

func (r queries) TouchLastLogin(ctx context.Context, userID string, at time.Time) error {
	_, err := r.q.Exec(ctx, `UPDATE users SET last_login_at = $2 WHERE id = $1`, userID, at)
	return err
}

func (r queries) UserIDByIdentity(ctx context.Context, provider, subject string) (string, error) {
	var id string
	err := r.q.QueryRow(ctx,
		`SELECT i.user_id FROM user_identities i JOIN users u ON u.id = i.user_id
		  WHERE i.provider = $1 AND i.subject = $2 AND u.deleted_at IS NULL`, provider, subject).Scan(&id)
	return id, notFound(err)
}

func (r queries) LinkIdentity(ctx context.Context, userID, provider, subject, email string, at time.Time) error {
	_, err := r.q.Exec(ctx,
		`INSERT INTO user_identities (user_id, provider, subject, email, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $5)`, userID, provider, subject, email, at)
	return err
}

func (r queries) IdentityProviders(ctx context.Context, userID string) ([]string, error) {
	var providers []string
	err := r.q.QueryRow(ctx,
		`SELECT COALESCE(array_agg(DISTINCT provider ORDER BY provider), '{}') FROM user_identities WHERE user_id = $1`,
		userID).Scan(&providers)
	return providers, err
}

func (r queries) CodeStats(ctx context.Context, email string, since time.Time) (*time.Time, int, error) {
	var last *time.Time
	var sent int
	err := r.q.QueryRow(ctx,
		`SELECT max(created_at), count(*) FILTER (WHERE created_at >= $2)
		   FROM email_verification_codes WHERE email = $1`, email, since).Scan(&last, &sent)
	return last, sent, err
}

func (r queries) CreateCode(ctx context.Context, c CodeRecord) error {
	_, err := r.q.Exec(ctx,
		`INSERT INTO email_verification_codes (email, code_hash, expires_at, created_at) VALUES ($1, $2, $3, $4)`,
		c.Email, c.Hash, c.ExpiresAt, c.CreatedAt)
	return err
}

// LatestCode locks the newest unused code of the email, so concurrent guesses are counted one
// after another.
func (r queries) LatestCode(ctx context.Context, email string) (StoredCode, error) {
	var c StoredCode
	err := r.q.QueryRow(ctx,
		`SELECT id, code_hash, expires_at, attempts, used_at FROM email_verification_codes
		  WHERE email = $1 AND used_at IS NULL ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, email).
		Scan(&c.ID, &c.Hash, &c.ExpiresAt, &c.Attempts, &c.UsedAt)
	return c, notFound(err)
}

func (r queries) IncrementCodeAttempts(ctx context.Context, codeID string) error {
	return mustAffect(r.q.Exec(ctx,
		`UPDATE email_verification_codes SET attempts = attempts + 1 WHERE id = $1`, codeID))
}

func (r queries) MarkCodeUsed(ctx context.Context, codeID string, at time.Time) error {
	return mustAffect(r.q.Exec(ctx,
		`UPDATE email_verification_codes SET used_at = $2 WHERE id = $1 AND used_at IS NULL`, codeID, at))
}

func (r queries) CreateSession(ctx context.Context, n NewSession) (Session, error) {
	var ip *netip.Addr
	if a, err := netip.ParseAddr(n.Client.IP); err == nil {
		ip = &a
	}
	var ua *string
	if n.Client.UserAgent != "" {
		ua = &n.Client.UserAgent
	}
	platform := n.Client.Platform
	if platform == "" {
		platform = "unknown"
	}
	var s Session
	err := r.q.QueryRow(ctx,
		`INSERT INTO auth_sessions (user_id, remember, platform, user_agent, ip_address, expires_at,
		                            last_used_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $7)
		 RETURNING id, user_id, expires_at, remember, revoked_at, last_used_at`,
		n.UserID, n.Remember, platform, ua, ip, n.ExpiresAt, n.CreatedAt).
		Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.Remember, &s.RevokedAt, &s.LastUsedAt)
	return s, err
}

func (r queries) SessionByID(ctx context.Context, id string) (Session, error) {
	var s Session
	err := r.q.QueryRow(ctx,
		`SELECT id, user_id, expires_at, remember, revoked_at, last_used_at FROM auth_sessions WHERE id = $1`, id).
		Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.Remember, &s.RevokedAt, &s.LastUsedAt)
	return s, notFound(err)
}

// RefreshTokenByHash locks the token row: two refreshes with the same token are serialised, so
// only one can rotate it.
func (r queries) RefreshTokenByHash(ctx context.Context, hash string) (RefreshToken, error) {
	var t RefreshToken
	err := r.q.QueryRow(ctx,
		`SELECT session_id, used_at FROM auth_refresh_tokens WHERE token_hash = $1 FOR UPDATE`, hash).
		Scan(&t.SessionID, &t.UsedAt)
	return t, notFound(err)
}

func (r queries) AddRefreshToken(ctx context.Context, sessionID, hash string, at time.Time) error {
	_, err := r.q.Exec(ctx,
		`INSERT INTO auth_refresh_tokens (token_hash, session_id, created_at) VALUES ($1, $2, $3)`, hash, sessionID, at)
	return err
}

func (r queries) MarkRefreshTokenUsed(ctx context.Context, hash string, at time.Time) error {
	return mustAffect(r.q.Exec(ctx,
		`UPDATE auth_refresh_tokens SET used_at = $2 WHERE token_hash = $1 AND used_at IS NULL`, hash, at))
}

func (r queries) TouchSession(ctx context.Context, sessionID string, at time.Time) error {
	_, err := r.q.Exec(ctx, `UPDATE auth_sessions SET last_used_at = $2 WHERE id = $1`, sessionID, at)
	return err
}

func (r queries) RevokeSession(ctx context.Context, sessionID string, at time.Time) error {
	_, err := r.q.Exec(ctx,
		`UPDATE auth_sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`, sessionID, at)
	return err
}

func (r queries) RevokeUserSessions(ctx context.Context, userID string, at time.Time) error {
	_, err := r.q.Exec(ctx,
		`UPDATE auth_sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`, userID, at)
	return err
}

func (r queries) CreateResetToken(ctx context.Context, userID, hash string, expiresAt, at time.Time) error {
	if _, err := r.q.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = $2 WHERE user_id = $1 AND used_at IS NULL`, userID, at); err != nil {
		return fmt.Errorf("supersede reset tokens: %w", err)
	}
	_, err := r.q.Exec(ctx,
		`INSERT INTO password_reset_tokens (token_hash, user_id, expires_at, created_at) VALUES ($1, $2, $3, $4)`,
		hash, userID, expiresAt, at)
	return err
}

func (r queries) ResetTokenByHash(ctx context.Context, hash string) (ResetToken, error) {
	var t ResetToken
	err := r.q.QueryRow(ctx,
		`SELECT user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = $1 FOR UPDATE`, hash).
		Scan(&t.UserID, &t.ExpiresAt, &t.UsedAt)
	return t, notFound(err)
}

func (r queries) MarkResetTokenUsed(ctx context.Context, hash string, at time.Time) error {
	return mustAffect(r.q.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = $2 WHERE token_hash = $1 AND used_at IS NULL`, hash, at))
}
