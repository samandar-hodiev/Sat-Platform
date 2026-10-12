package migrate

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/samandar-hodiev/sat-platform/api/migrations"
)

// These tests run only against a disposable database (TEST_DATABASE_URL). They revert every
// migration first, so never point them at a database with data you want to keep.

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	return url
}

func newMigrator(t *testing.T, url string) *Migrator {
	t.Helper()
	mg, err := New(url, migrations.FS, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mg.Close() })
	return mg
}

func tableExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var ok bool
	err := pool.QueryRow(context.Background(), "SELECT to_regclass($1) IS NOT NULL", "public."+name).Scan(&ok)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

func TestMigrationsUpDownUp(t *testing.T) {
	url := testDatabaseURL(t)
	mg := newMigrator(t, url)
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	tables := []string{"users", "profiles", "user_preferences", "audit_logs"}

	if err := mg.DownAll(); err != nil {
		t.Fatalf("initial down: %v", err)
	}
	if err := mg.Up(); err != nil {
		t.Fatalf("up: %v", err)
	}
	st, err := mg.Status()
	if err != nil {
		t.Fatal(err)
	}
	if st.Dirty || st.Version != st.Latest || len(st.Pending) != 0 {
		t.Fatalf("after up: %+v", st)
	}
	for _, tbl := range tables {
		if !tableExists(t, pool, tbl) {
			t.Fatalf("table %s missing after up", tbl)
		}
	}

	// Re-running up on an up-to-date database is a no-op and keeps data.
	if _, err := pool.Exec(context.Background(), "INSERT INTO users (email) VALUES ('keep@example.com')"); err != nil {
		t.Fatal(err)
	}
	if err := mg.Up(); err != nil {
		t.Fatalf("second up: %v", err)
	}
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM users").Scan(&n); err != nil || n != 1 {
		t.Fatalf("data changed by second up: %d, %v", n, err)
	}

	// Step down one by one, then all the way: every down must succeed.
	if err := mg.Down(1); err != nil {
		t.Fatalf("down 1: %v", err)
	}
	// Exactly the newest migration was reverted (later sessions keep adding migrations, so the
	// check is by version, not by a table name).
	versions, err := Versions(migrations.FS)
	if err != nil || len(versions) < 2 {
		t.Fatalf("versions: %v %v", versions, err)
	}
	if st, err := mg.Status(); err != nil || st.Dirty || st.Version != versions[len(versions)-2] {
		t.Fatalf("after down 1: %+v, %v", st, err)
	}
	if err := mg.DownAll(); err != nil {
		t.Fatalf("down all: %v", err)
	}
	for _, tbl := range tables {
		if tableExists(t, pool, tbl) {
			t.Fatalf("table %s left after down", tbl)
		}
	}
	var fn bool
	if err := pool.QueryRow(context.Background(), "SELECT to_regproc('set_updated_at') IS NOT NULL").Scan(&fn); err != nil || fn {
		t.Fatalf("set_updated_at left after down: %v", err)
	}
	st, err = mg.Status()
	if err != nil || st.Version != 0 || len(st.Pending) == 0 {
		t.Fatalf("status after down: %+v, %v", st, err)
	}

	if err := mg.Up(); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

// TestSchemaConstraints checks the invariants the database enforces on its own.
func TestSchemaConstraints(t *testing.T) {
	url := testDatabaseURL(t)
	mg := newMigrator(t, url)
	if err := mg.DownAll(); err != nil {
		t.Fatal(err)
	}
	if err := mg.Up(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	exec := func(sql string, args ...any) error {
		_, err := pool.Exec(ctx, sql, args...)
		return err
	}
	mustExec := func(t *testing.T, sql string, args ...any) {
		t.Helper()
		if err := exec(sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	wantCode := func(t *testing.T, err error, code string) {
		t.Helper()
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != code {
			t.Fatalf("want SQLSTATE %s, got %v", code, err)
		}
	}
	const (
		checkViolation  = "23514"
		uniqueViolation = "23505"
		fkViolation     = "23503"
		insufficientPri = "42501"
	)
	newUser := func(t *testing.T, email string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, "INSERT INTO users (email) VALUES ($1) RETURNING id", email).Scan(&id); err != nil {
			t.Fatalf("insert user %s: %v", email, err)
		}
		return id
	}

	t.Run("users defaults", func(t *testing.T) {
		id := newUser(t, "defaults@example.com")
		var role, status string
		var hash *string
		err := pool.QueryRow(ctx, "SELECT role, status, password_hash FROM users WHERE id = $1", id).Scan(&role, &status, &hash)
		if err != nil || role != "learner" || status != "active" || hash != nil {
			t.Fatalf("got role=%q status=%q hash=%v err=%v", role, status, hash, err)
		}
	})

	t.Run("users email must be normalised", func(t *testing.T) {
		for _, email := range []string{"Upper@Example.com", " space@example.com", "no-at-sign", "@example.com"} {
			wantCode(t, exec("INSERT INTO users (email) VALUES ($1)", email), checkViolation)
		}
	})

	t.Run("users role and status allow-list", func(t *testing.T) {
		wantCode(t, exec("INSERT INTO users (email, role) VALUES ('r@example.com', 'admin')"), checkViolation)
		wantCode(t, exec("INSERT INTO users (email, status) VALUES ('s@example.com', 'banned')"), checkViolation)
		mustExec(t, "INSERT INTO users (email, role) VALUES ('owner@example.com', 'owner')")
	})

	t.Run("users email unique among live accounts", func(t *testing.T) {
		id := newUser(t, "dup@example.com")
		wantCode(t, exec("INSERT INTO users (email) VALUES ('dup@example.com')"), uniqueViolation)
		// A deleted account frees the email for a new sign-up.
		mustExec(t, "UPDATE users SET deleted_at = now() WHERE id = $1", id)
		newUser(t, "dup@example.com")
		// …and a second live account with it is still rejected.
		wantCode(t, exec("INSERT INTO users (email) VALUES ('dup@example.com')"), uniqueViolation)
	})

	t.Run("updated_at trigger", func(t *testing.T) {
		id := newUser(t, "touch@example.com")
		mustExec(t, "UPDATE users SET created_at = created_at - interval '1 hour', updated_at = updated_at - interval '1 hour' WHERE id = $1", id)
		// The trigger overwrote updated_at above too; a later change must move it past created_at.
		mustExec(t, "UPDATE users SET last_login_at = now() WHERE id = $1", id)
		var created, updated time.Time
		if err := pool.QueryRow(ctx, "SELECT created_at, updated_at FROM users WHERE id = $1", id).Scan(&created, &updated); err != nil {
			t.Fatal(err)
		}
		if !updated.After(created) {
			t.Fatalf("updated_at %v not after created_at %v", updated, created)
		}

		mustExec(t, "INSERT INTO profiles (user_id) VALUES ($1)", id)
		mustExec(t, "INSERT INTO user_preferences (user_id) VALUES ($1)", id)
		for _, tbl := range []string{"profiles", "user_preferences"} {
			var fired bool
			err := pool.QueryRow(ctx,
				"UPDATE "+tbl+" SET created_at = created_at - interval '1 hour' WHERE user_id = $1 RETURNING updated_at > created_at", id).Scan(&fired)
			if err != nil || !fired {
				t.Fatalf("%s: updated_at trigger did not run (%v)", tbl, err)
			}
		}
	})

	t.Run("profiles constraints", func(t *testing.T) {
		id := newUser(t, "profile@example.com")
		mustExec(t, `INSERT INTO profiles (user_id, first_name, last_name, phone, date_of_birth, timezone)
		             VALUES ($1, 'Ali', 'Valiyev', '+998901234567', '2008-05-01', 'Asia/Tashkent')`, id)
		wantCode(t, exec("INSERT INTO profiles (user_id) VALUES ($1)", id), uniqueViolation)
		wantCode(t, exec("UPDATE profiles SET phone = '998901234567' WHERE user_id = $1", id), checkViolation)
		wantCode(t, exec("UPDATE profiles SET first_name = '   ' WHERE user_id = $1", id), checkViolation)
		wantCode(t, exec("UPDATE profiles SET date_of_birth = '1800-01-01' WHERE user_id = $1", id), checkViolation)
		wantCode(t, exec("INSERT INTO profiles (user_id) VALUES (gen_random_uuid())"), fkViolation)
	})

	t.Run("user_preferences defaults and allow-lists", func(t *testing.T) {
		id := newUser(t, "prefs@example.com")
		mustExec(t, "INSERT INTO user_preferences (user_id) VALUES ($1)", id)
		var theme, accent, density, motion string
		var collapsed bool
		err := pool.QueryRow(ctx, "SELECT theme, accent, density, motion, sidebar_collapsed FROM user_preferences WHERE user_id = $1", id).
			Scan(&theme, &accent, &density, &motion, &collapsed)
		if err != nil || theme != "light" || accent != "neutral" || density != "comfortable" || motion != "system" || collapsed {
			t.Fatalf("defaults: %s %s %s %s %v (%v)", theme, accent, density, motion, collapsed, err)
		}
		for col, bad := range map[string]string{"theme": "sepia", "accent": "orange", "density": "tight", "motion": "off"} {
			wantCode(t, exec("UPDATE user_preferences SET "+col+" = $2 WHERE user_id = $1", id, bad), checkViolation)
		}
	})

	t.Run("deleting a user cascades to owned rows", func(t *testing.T) {
		id := newUser(t, "cascade@example.com")
		mustExec(t, "INSERT INTO profiles (user_id) VALUES ($1)", id)
		mustExec(t, "INSERT INTO user_preferences (user_id) VALUES ($1)", id)
		mustExec(t, "DELETE FROM users WHERE id = $1", id)
		var n int
		err := pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM profiles WHERE user_id = $1) + (SELECT count(*) FROM user_preferences WHERE user_id = $1)", id).Scan(&n)
		if err != nil || n != 0 {
			t.Fatalf("rows left: %d (%v)", n, err)
		}
	})

	t.Run("audit_logs constraints", func(t *testing.T) {
		mustExec(t, `INSERT INTO audit_logs (action, entity_type, entity_id, metadata, ip_address)
		             VALUES ('question.publish', 'question', 'abc', '{"from":"draft"}', '203.0.113.7')`)
		wantCode(t, exec("INSERT INTO audit_logs (action) VALUES ('Publish')"), checkViolation)
		wantCode(t, exec("INSERT INTO audit_logs (action) VALUES ('publish')"), checkViolation)
		wantCode(t, exec("INSERT INTO audit_logs (action, metadata) VALUES ('a.b', '[]')"), checkViolation)
		wantCode(t, exec("INSERT INTO audit_logs (action, entity_type) VALUES ('a.b', 'Bad Type')"), checkViolation)
	})

	t.Run("audit_logs is append-only", func(t *testing.T) {
		actor := newUser(t, "auditor@example.com")
		var logID string
		err := pool.QueryRow(ctx,
			"INSERT INTO audit_logs (actor_user_id, actor_role, action) VALUES ($1, 'owner', 'learner.suspend') RETURNING id", actor).Scan(&logID)
		if err != nil {
			t.Fatal(err)
		}
		wantCode(t, exec("UPDATE audit_logs SET action = 'learner.restore' WHERE id = $1", logID), insufficientPri)
		wantCode(t, exec("UPDATE audit_logs SET actor_user_id = NULL, action = 'x.y' WHERE id = $1", logID), insufficientPri)
		wantCode(t, exec("DELETE FROM audit_logs WHERE id = $1", logID), insufficientPri)
		wantCode(t, exec("TRUNCATE audit_logs"), insufficientPri)

		// Deleting the actor is allowed: the entry stays, only actor_user_id becomes NULL.
		mustExec(t, "DELETE FROM users WHERE id = $1", actor)
		var gotActor *string
		var role, action string
		err = pool.QueryRow(ctx, "SELECT actor_user_id, actor_role, action FROM audit_logs WHERE id = $1", logID).Scan(&gotActor, &role, &action)
		if err != nil || gotActor != nil || role != "owner" || action != "learner.suspend" {
			t.Fatalf("after actor delete: actor=%v role=%q action=%q err=%v", gotActor, role, action, err)
		}
	})
}
