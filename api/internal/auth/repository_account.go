package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL side of the account changes of My Profile (2.10, account.go). Tables: users,
// user_identities, auth_sessions, password_reset_tokens and 2.10's account_action_codes.

func (r queries) ActionCodeStats(ctx context.Context, userID, purpose string, since time.Time) (*time.Time, int, error) {
	var last *time.Time
	var sent int
	err := r.q.QueryRow(ctx,
		`SELECT max(created_at), count(*) FILTER (WHERE created_at >= $3)
		   FROM account_action_codes WHERE user_id = $1 AND purpose = $2`, userID, purpose, since).Scan(&last, &sent)
	return last, sent, notFound(err)
}

func (r queries) CreateActionCode(ctx context.Context, c ActionCodeRecord) error {
	_, err := r.q.Exec(ctx,
		`INSERT INTO account_action_codes (user_id, purpose, email, code_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		c.UserID, c.Purpose, c.Email, c.Hash, c.ExpiresAt, c.CreatedAt)
	return err
}

// LatestActionCode locks the newest unused code, so concurrent guesses are counted one after
// another.
func (r queries) LatestActionCode(ctx context.Context, userID, purpose string) (StoredActionCode, error) {
	var c StoredActionCode
	err := r.q.QueryRow(ctx,
		`SELECT id, email, code_hash, expires_at, attempts, used_at FROM account_action_codes
		  WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL
		  ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, userID, purpose).
		Scan(&c.ID, &c.Email, &c.Hash, &c.ExpiresAt, &c.Attempts, &c.UsedAt)
	return c, notFound(err)
}

func (r queries) IncrementActionCodeAttempts(ctx context.Context, codeID string) error {
	return mustAffect(r.q.Exec(ctx,
		`UPDATE account_action_codes SET attempts = attempts + 1 WHERE id = $1`, codeID))
}

func (r queries) MarkActionCodeUsed(ctx context.Context, codeID string, at time.Time) error {
	return mustAffect(r.q.Exec(ctx,
		`UPDATE account_action_codes SET used_at = $2 WHERE id = $1 AND used_at IS NULL`, codeID, at))
}

func (r queries) ChangeEmail(ctx context.Context, userID, email string, at time.Time) error {
	err := mustAffect(r.q.Exec(ctx,
		`UPDATE users SET email = $2, email_verified_at = $3 WHERE id = $1 AND deleted_at IS NULL`, userID, email, at))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrEmailTaken
	}
	return notFound(err)
}

func (r queries) Identity(ctx context.Context, userID, provider string) (Identity, error) {
	var id Identity
	err := r.q.QueryRow(ctx,
		`SELECT provider, email, created_at FROM user_identities WHERE user_id = $1 AND provider = $2
		  ORDER BY created_at LIMIT 1`, userID, provider).Scan(&id.Provider, &id.Email, &id.LinkedAt)
	return id, notFound(err)
}

func (r queries) UnlinkIdentity(ctx context.Context, userID, provider string) error {
	return notFound(mustAffect(r.q.Exec(ctx,
		`DELETE FROM user_identities WHERE user_id = $1 AND provider = $2`, userID, provider)))
}

func (r queries) RevokeOtherSessions(ctx context.Context, userID, keepSessionID string, at time.Time) error {
	_, err := r.q.Exec(ctx,
		`UPDATE auth_sessions SET revoked_at = $3
		  WHERE user_id = $1 AND id::text <> $2 AND revoked_at IS NULL`, userID, keepSessionID, at)
	return err
}

// DeletedEmailDomain is the placeholder domain of deleted accounts' emails. ".invalid" is
// reserved (RFC 2606), so the placeholder can never receive mail or collide with a real address.
const DeletedEmailDomain = "deleted.invalid"

func (r queries) DeleteUser(ctx context.Context, userID string, at time.Time) error {
	if err := notFound(mustAffect(r.q.Exec(ctx,
		`UPDATE users SET deleted_at = $2, email = 'deleted-' || id::text || '@`+DeletedEmailDomain+`',
		                  password_hash = NULL, email_verified_at = NULL
		  WHERE id = $1 AND deleted_at IS NULL`, userID, at))); err != nil {
		return err
	}
	for _, q := range []string{
		`DELETE FROM user_identities WHERE user_id = $1`,
		`DELETE FROM password_reset_tokens WHERE user_id = $1`,
		`DELETE FROM account_action_codes WHERE user_id = $1`,
	} {
		if _, err := r.q.Exec(ctx, q, userID); err != nil {
			return err
		}
	}
	return r.RevokeUserSessions(ctx, userID, at)
}
