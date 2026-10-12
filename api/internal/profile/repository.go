package profile

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements Store on the profiles table.
type PostgresStore struct {
	db *pgxpool.Pool
}

func NewPostgresStore(db *pgxpool.Pool) *PostgresStore { return &PostgresStore{db: db} }

const profileColumns = `user_id, COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(phone, ''),
	date_of_birth, COALESCE(avatar_key, ''), timezone, profile_completed_at, updated_at`

func scanProfile(row pgx.Row) (Profile, error) {
	var p Profile
	err := row.Scan(&p.UserID, &p.FirstName, &p.LastName, &p.Phone, &p.DateOfBirth, &p.AvatarKey,
		&p.Timezone, &p.CompletedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	return p, err
}

func (s *PostgresStore) Get(ctx context.Context, userID string) (Profile, error) {
	p, err := scanProfile(s.db.QueryRow(ctx, `SELECT `+profileColumns+` FROM profiles WHERE user_id = $1`, userID))
	return p, mapErr(err)
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *PostgresStore) CompleteSetup(ctx context.Context, userID string, in Setup, now time.Time, also func(pgx.Tx) error) (Profile, error) {
	var p Profile
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		// The WHERE makes one statement decide: a finished profile is never overwritten, even
		// by two submits racing.
		var err error
		p, err = scanProfile(tx.QueryRow(ctx, `
		INSERT INTO profiles (user_id, first_name, last_name, phone, date_of_birth, timezone, profile_completed_at)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6, 'UTC'), $7)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name           = EXCLUDED.first_name,
			last_name            = EXCLUDED.last_name,
			phone                = EXCLUDED.phone,
			date_of_birth        = EXCLUDED.date_of_birth,
			timezone             = COALESCE($6, profiles.timezone),
			profile_completed_at = EXCLUDED.profile_completed_at
		WHERE profiles.profile_completed_at IS NULL
		RETURNING `+profileColumns,
			userID, in.FirstName, in.LastName, nullable(in.Phone), in.DateOfBirth, nullable(in.Timezone), now))
		if errors.Is(err, ErrNotFound) {
			return ErrAlreadyCompleted
		}
		if err != nil || also == nil {
			return err
		}
		return also(tx)
	})
	if err != nil {
		return Profile{}, mapErr(err)
	}
	return p, nil
}

func (s *PostgresStore) Prefill(ctx context.Context, userID, firstName, lastName string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, last_name) VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = COALESCE(profiles.first_name, EXCLUDED.first_name),
			last_name  = COALESCE(profiles.last_name, EXCLUDED.last_name)
		WHERE profiles.profile_completed_at IS NULL`,
		userID, nullable(firstName), nullable(lastName))
	return mapErr(err)
}

func (s *PostgresStore) SetAvatar(ctx context.Context, userID, key string) (string, error) {
	return s.swapAvatar(ctx, userID, &key)
}

func (s *PostgresStore) ClearAvatar(ctx context.Context, userID string) (string, error) {
	return s.swapAvatar(ctx, userID, nil)
}

// swapAvatar replaces avatar_key and returns the old value; the row lock makes concurrent
// uploads each see the key they replaced, so no file is left behind.
func (s *PostgresStore) swapAvatar(ctx context.Context, userID string, key *string) (string, error) {
	var previous string
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO profiles (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT COALESCE(avatar_key, '') FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).
			Scan(&previous); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE profiles SET avatar_key = $2 WHERE user_id = $1`, userID, key)
		return err
	})
	return previous, mapErr(err)
}

// mapErr turns "the user does not exist" (FK violation, invalid UUID) into ErrNotFound.
func mapErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02") {
		return ErrNotFound
	}
	return err
}
