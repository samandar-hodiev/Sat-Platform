package preferences

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements Store on the user_preferences table.
type PostgresStore struct {
	db *pgxpool.Pool
}

func NewPostgresStore(db *pgxpool.Pool) *PostgresStore { return &PostgresStore{db: db} }

const prefsColumns = `theme, accent, density, motion, sidebar_collapsed,
	wallpaper_light_source, COALESCE(wallpaper_light_gallery_id, ''), COALESCE(wallpaper_light_key, ''),
	wallpaper_dark_source, COALESCE(wallpaper_dark_gallery_id, ''), COALESCE(wallpaper_dark_key, ''),
	wallpaper_opacity, wallpaper_blur, updated_at`

func scanPrefs(row pgx.Row) (Prefs, error) {
	var p Prefs
	var opacity, blur int16
	err := row.Scan(&p.Theme, &p.Accent, &p.Density, &p.Motion, &p.SidebarCollapsed,
		&p.Light.Source, &p.Light.GalleryID, &p.Light.Key,
		&p.Dark.Source, &p.Dark.GalleryID, &p.Dark.Key,
		&opacity, &blur, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Prefs{}, ErrNotFound
	}
	p.Opacity, p.Blur = int(opacity), int(blur)
	return p, err
}

func (s *PostgresStore) Get(ctx context.Context, userID string) (Prefs, error) {
	p, err := scanPrefs(s.db.QueryRow(ctx, `SELECT `+prefsColumns+` FROM user_preferences WHERE user_id = $1`, userID))
	return p, mapErr(err)
}

// locked runs fn on the learner's row, created from the defaults if missing, under FOR UPDATE:
// concurrent changes see each other's result and uploads each see the key they replaced.
func (s *PostgresStore) locked(ctx context.Context, userID string, fn func(Prefs) (Prefs, error)) (Prefs, error) {
	var out Prefs
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO user_preferences (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
			return err
		}
		current, err := scanPrefs(tx.QueryRow(ctx, `SELECT `+prefsColumns+` FROM user_preferences WHERE user_id = $1 FOR UPDATE`, userID))
		if err != nil {
			return err
		}
		next, err := fn(current)
		if err != nil {
			return err
		}
		out, err = scanPrefs(tx.QueryRow(ctx, `UPDATE user_preferences SET
				theme = $2, accent = $3, density = $4, motion = $5, sidebar_collapsed = $6,
				wallpaper_light_source = $7, wallpaper_light_gallery_id = $8, wallpaper_light_key = $9,
				wallpaper_dark_source = $10, wallpaper_dark_gallery_id = $11, wallpaper_dark_key = $12,
				wallpaper_opacity = $13, wallpaper_blur = $14
			WHERE user_id = $1 RETURNING `+prefsColumns,
			userID, next.Theme, next.Accent, next.Density, next.Motion, next.SidebarCollapsed,
			next.Light.Source, nullable(next.Light.GalleryID), nullable(next.Light.Key),
			next.Dark.Source, nullable(next.Dark.GalleryID), nullable(next.Dark.Key),
			next.Opacity, next.Blur))
		return err
	})
	return out, mapErr(err)
}

func (s *PostgresStore) Update(ctx context.Context, userID string, patch Patch) (Prefs, error) {
	return s.locked(ctx, userID, patch.Apply)
}

func (s *PostgresStore) SetWallpaperKey(ctx context.Context, userID string, mode Mode, key string) (Prefs, string, error) {
	var previous string
	p, err := s.locked(ctx, userID, func(p Prefs) (Prefs, error) {
		previous = setKey(&p, mode, key)
		return p, nil
	})
	return p, previous, err
}

func (s *PostgresStore) ClearWallpaperKey(ctx context.Context, userID string, mode Mode) (Prefs, string, error) {
	var previous string
	p, err := s.locked(ctx, userID, func(p Prefs) (Prefs, error) {
		previous = setKey(&p, mode, "")
		return p, nil
	})
	return p, previous, err
}

// setKey stores (or with "" removes) the uploaded image of mode and returns the old key. A new
// image is selected; removing the selected image falls back to no wallpaper.
func setKey(p *Prefs, mode Mode, key string) string {
	w := &p.Light
	if mode == ModeDark {
		w = &p.Dark
	}
	previous := w.Key
	w.Key = key
	switch {
	case key != "":
		w.Source, w.GalleryID = SourceCustom, ""
	case w.Source == SourceCustom:
		w.Source = SourceNone
	}
	return previous
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// mapErr: a missing user (FK violation on insert, or a malformed id) is ErrNotFound.
func mapErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02") {
		return ErrNotFound
	}
	return err
}
