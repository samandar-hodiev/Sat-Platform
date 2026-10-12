package migrate

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/samandar-hodiev/sat-platform/api/migrations"
)

// The real migration folder must always pass validation: every file well named, every up
// with its down, no duplicate versions.
func TestRepositoryMigrationsAreValid(t *testing.T) {
	versions, err := Versions(migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) == 0 {
		t.Fatal("no migrations found")
	}
	for i := 1; i < len(versions); i++ {
		if versions[i] <= versions[i-1] {
			t.Fatalf("versions not ascending: %v", versions)
		}
	}
}

func TestRepositoryMigrationsAreTransactional(t *testing.T) {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		body, err := migrations.FS.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		sql := string(body)
		if strings.Contains(strings.ToUpper(sql), "CONCURRENTLY") {
			continue // cannot run in a transaction (docs/DB.md)
		}
		if !strings.Contains(sql, "BEGIN;") || !strings.Contains(sql, "COMMIT;") {
			t.Errorf("%s: wrap the statements in BEGIN; … COMMIT;", e.Name())
		}
	}
}

func TestVersionsRejectsBadFiles(t *testing.T) {
	file := &fstest.MapFile{Data: []byte("BEGIN; COMMIT;")}
	cases := map[string]fstest.MapFS{
		"missing down": {"20261012090000_a.up.sql": file},
		"missing up":   {"20261012090000_a.down.sql": file},
		"bad name":     {"0001_a.up.sql": file, "0001_a.down.sql": file},
		"camel case":   {"20261012090000_addThing.up.sql": file, "20261012090000_addThing.down.sql": file},
		"bad time":     {"20261399090000_a.up.sql": file, "20261399090000_a.down.sql": file},
		"stray file":   {"notes.txt": file},
		"duplicate version": {
			"20261012090000_a.up.sql": file, "20261012090000_a.down.sql": file,
			"20261012090000_b.up.sql": file, "20261012090000_b.down.sql": file,
		},
	}
	for name, files := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Versions(files); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestVersionsSortsAndIgnoresDocs(t *testing.T) {
	file := &fstest.MapFile{Data: []byte("BEGIN; COMMIT;")}
	files := fstest.MapFS{
		"README.md":               file,
		"embed.go":                file,
		"20261012090100_b.up.sql": file, "20261012090100_b.down.sql": file,
		"20261012090000_a.up.sql": file, "20261012090000_a.down.sql": file,
	}
	got, err := Versions(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 20261012090000 || got[1] != 20261012090100 {
		t.Fatalf("got %v", got)
	}
}

func TestNewRejectsBadURLWithoutEchoingIt(t *testing.T) {
	_, err := New("postgres://user:s3cret@%zz/db", migrations.FS, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "s3cret") {
		t.Errorf("error leaks the connection string: %v", err)
	}
}

func TestDownRejectsNonPositive(t *testing.T) {
	mg := &Migrator{}
	if err := mg.Down(0); err == nil {
		t.Fatal("down 0 accepted")
	}
}
