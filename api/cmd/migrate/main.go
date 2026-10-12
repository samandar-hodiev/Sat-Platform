// Command migrate applies the database migrations in api/migrations (embedded at build time).
//
//	go run ./cmd/migrate up            apply every pending migration
//	go run ./cmd/migrate down [n]      revert the last n migrations (default 1)
//	go run ./cmd/migrate status        show the applied version and pending migrations
//	go run ./cmd/migrate force <v>     mark version v as applied after a manual repair
//	go run ./cmd/migrate create <name> create an empty up/down pair in ./migrations
//
// The database comes from DATABASE_URL (root .env in development, like the API server).
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/samandar-hodiev/sat-platform/api/internal/config"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/logger"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/migrate"
	"github.com/samandar-hodiev/sat-platform/api/migrations"
)

const usage = `usage: migrate <command>

commands:
  up             apply every pending migration
  down [n]       revert the last n migrations (default 1)
  status         show the applied version and pending migrations
  force <v>      set the version after repairing a failed migration (-1 = none)
  create <name>  create YYYYMMDDHHMMSS_<name>.up.sql/.down.sql in ./migrations`

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	cmd, rest := args[0], args[1:]

	if cmd == "create" {
		if len(rest) != 1 {
			return errors.New("create needs exactly one name, e.g. create add_learner_goals")
		}
		up, down, err := create("migrations", rest[0], time.Now().UTC())
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "created %s\ncreated %s\n", up, down)
		return nil
	}

	dbURL, err := config.DatabaseURL()
	if err != nil {
		return err
	}
	mg, err := migrate.New(dbURL, migrations.FS, logger.New("info", "text"))
	if err != nil {
		return err
	}
	defer mg.Close()

	switch cmd {
	case "up":
		if len(rest) != 0 {
			return errors.New("up takes no arguments")
		}
		if err := mg.Up(); err != nil {
			return err
		}
	case "down":
		n := 1
		if len(rest) > 1 {
			return errors.New("down takes at most one argument")
		}
		if len(rest) == 1 {
			if n, err = strconv.Atoi(rest[0]); err != nil || n <= 0 {
				return errors.New("down: n must be a positive number")
			}
		}
		if err := mg.Down(n); err != nil {
			return err
		}
	case "force":
		if len(rest) != 1 {
			return errors.New("force needs a version")
		}
		v, err := strconv.Atoi(rest[0])
		if err != nil || v < -1 {
			return errors.New("force: version must be a migration version or -1")
		}
		if err := mg.Force(v); err != nil {
			return err
		}
	case "status":
		if len(rest) != 0 {
			return errors.New("status takes no arguments")
		}
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
	return printStatus(out, mg)
}

func printStatus(out io.Writer, mg *migrate.Migrator) error {
	st, err := mg.Status()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "applied version: %d\n", st.Version)
	fmt.Fprintf(out, "latest version:  %d\n", st.Latest)
	if st.Dirty {
		fmt.Fprintln(out, "DIRTY: the last migration failed half-way; repair the database, then run force <version>")
	}
	fmt.Fprintf(out, "pending:         %d\n", len(st.Pending))
	for _, v := range st.Pending {
		fmt.Fprintf(out, "  %d\n", v)
	}
	return nil
}

var namePattern = regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`)

// create writes an empty migration pair. The timestamp prefix keeps sessions working in
// parallel from picking the same version.
func create(dir, name string, now time.Time) (string, string, error) {
	name = strings.TrimSpace(name)
	if !namePattern.MatchString(name) || len(name) > 80 {
		return "", "", errors.New("name must be snake_case (a-z, 0-9, _), e.g. create_learner_goals")
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", "", fmt.Errorf("%s not found: run from the api/ directory", dir)
	}
	base := filepath.Join(dir, now.Format("20060102150405")+"_"+name)
	up, down := base+".up.sql", base+".down.sql"
	for path, body := range map[string]string{
		up:   "BEGIN;\n\n-- TODO: schema change\n\nCOMMIT;\n",
		down: "BEGIN;\n\n-- TODO: exact reverse of the .up.sql\n\nCOMMIT;\n",
	} {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return "", "", err
		}
		_, werr := f.WriteString(body)
		if err := errors.Join(werr, f.Close()); err != nil {
			return "", "", err
		}
	}
	return up, down, nil
}
