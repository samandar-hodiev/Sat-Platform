// Package migrate applies the SQL migrations embedded from api/migrations with
// golang-migrate. It is used by cmd/migrate and by integration tests; the API server never
// migrates on start-up (running migrations is a separate deploy step).
//
// Applied versions are recorded in the schema_migrations table, so Up on an up-to-date
// database does nothing. golang-migrate takes a PostgreSQL advisory lock, so two runs at the
// same time cannot interleave.
package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// FileNamePattern is the required migration file name: a 14-digit timestamp, a snake_case
// name and the direction (docs/DB.md).
var FileNamePattern = regexp.MustCompile(`^(\d{14})_([a-z0-9]+(?:_[a-z0-9]+)*)\.(up|down)\.sql$`)

// Status describes the database schema state.
type Status struct {
	// Version is the last applied migration (0 when none has run).
	Version uint
	// Dirty means a migration failed half-way; fix the database by hand, then Force.
	Dirty bool
	// Latest is the newest migration available in the files.
	Latest uint
	// Pending lists available migrations newer than Version.
	Pending []uint
}

// Migrator runs migrations from a file system against one database.
type Migrator struct {
	m        *migrate.Migrate
	db       *sql.DB
	versions []uint
}

// New connects to databaseURL (a postgres:// URL, same as DATABASE_URL) and prepares the
// migrations found in files. Close must be called.
func New(databaseURL string, files fs.FS, log *slog.Logger) (*Migrator, error) {
	versions, err := Versions(files)
	if err != nil {
		return nil, err
	}

	connCfg, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		// pgx errors can include the URL (and its password); never return them.
		return nil, errors.New("parse database URL: invalid connection string")
	}
	connCfg.ConnectTimeout = 5 * time.Second
	db := stdlib.OpenDB(*connCfg)

	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{StatementTimeout: 5 * time.Minute})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	src, err := iofs.New(files, ".")
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init migrations: %w", err)
	}
	if log != nil {
		m.Log = logAdapter{log}
	}
	return &Migrator{m: m, db: db, versions: versions}, nil
}

// Close releases the database connection.
func (mg *Migrator) Close() error {
	srcErr, dbErr := mg.m.Close()
	closeErr := mg.db.Close()
	return errors.Join(srcErr, dbErr, closeErr)
}

// Up applies every pending migration. It returns nil when there is nothing to do.
func (mg *Migrator) Up() error {
	return ignoreNoChange(mg.m.Up())
}

// Down reverts the last n applied migrations (n must be positive).
func (mg *Migrator) Down(n int) error {
	if n <= 0 {
		return errors.New("down: number of migrations must be positive")
	}
	return ignoreNoChange(mg.m.Steps(-n))
}

// DownAll reverts every migration. Used by integration tests on a disposable database.
func (mg *Migrator) DownAll() error {
	return ignoreNoChange(mg.m.Down())
}

// Force sets the recorded version without running SQL and clears the dirty flag. Use it only
// after repairing a failed migration by hand. version -1 means "no migration applied".
func (mg *Migrator) Force(version int) error {
	return mg.m.Force(version)
}

// Status reports the applied version and pending migrations.
func (mg *Migrator) Status() (Status, error) {
	var st Status
	v, dirty, err := mg.m.Version()
	switch {
	case errors.Is(err, migrate.ErrNilVersion):
	case err != nil:
		return st, err
	default:
		st.Version, st.Dirty = v, dirty
	}
	if n := len(mg.versions); n > 0 {
		st.Latest = mg.versions[n-1]
	}
	for _, fv := range mg.versions {
		if fv > st.Version {
			st.Pending = append(st.Pending, fv)
		}
	}
	return st, nil
}

// Versions validates the migration files and returns their versions in ascending order.
// Every version must have exactly one .up.sql and one .down.sql file with the same name, and
// no other files except Go sources and README.md may be present.
func Versions(files fs.FS) ([]uint, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	type pair struct {
		name     string
		up, down bool
	}
	byVersion := map[uint]*pair{}
	var problems []error
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == "README.md" || strings.HasSuffix(name, ".go") {
			continue
		}
		match := FileNamePattern.FindStringSubmatch(name)
		if match == nil {
			problems = append(problems, fmt.Errorf("%s: name must be YYYYMMDDHHMMSS_snake_case.(up|down).sql", name))
			continue
		}
		if _, err := time.Parse("20060102150405", match[1]); err != nil {
			problems = append(problems, fmt.Errorf("%s: prefix is not a valid timestamp", name))
			continue
		}
		var v uint
		_, _ = fmt.Sscan(match[1], &v)
		p := byVersion[v]
		if p == nil {
			p = &pair{name: match[2]}
			byVersion[v] = p
		}
		if p.name != match[2] {
			problems = append(problems, fmt.Errorf("%s: version %d is used by two different migrations", name, v))
			continue
		}
		if match[3] == "up" {
			p.up = true
		} else {
			p.down = true
		}
	}
	versions := make([]uint, 0, len(byVersion))
	for v, p := range byVersion {
		if !p.up || !p.down {
			problems = append(problems, fmt.Errorf("%d_%s: needs both .up.sql and .down.sql", v, p.name))
		}
		versions = append(versions, v)
	}
	if err := errors.Join(problems...); err != nil {
		return nil, err
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	return versions, nil
}

func ignoreNoChange(err error) error {
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}

type logAdapter struct{ log *slog.Logger }

func (l logAdapter) Printf(format string, v ...any) {
	l.log.Info(strings.TrimSpace(fmt.Sprintf(format, v...)))
}

func (l logAdapter) Verbose() bool { return false }
