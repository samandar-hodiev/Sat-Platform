package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/samandar-hodiev/sat-platform/api/internal/platform/migrate"
)

func TestCreateWritesValidPair(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 12, 9, 30, 15, 0, time.UTC)
	up, down, err := create(dir, "create_learner_goals", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{up, down} {
		name := filepath.Base(p)
		if !strings.HasPrefix(name, "20261012093015_create_learner_goals.") || !migrate.FileNamePattern.MatchString(name) {
			t.Errorf("bad file name %s", name)
		}
		body, err := os.ReadFile(p)
		if err != nil || !strings.Contains(string(body), "BEGIN;") {
			t.Errorf("%s: %q, %v", name, body, err)
		}
	}
	versions, err := migrate.Versions(os.DirFS(dir))
	if err != nil || len(versions) != 1 {
		t.Fatalf("created pair does not validate: %v %v", versions, err)
	}
	// Never overwrite an existing migration.
	if _, _, err := create(dir, "create_learner_goals", now); err == nil {
		t.Fatal("existing files overwritten")
	}
}

func TestCreateRejectsBadNames(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"", "AddGoals", "add-goals", "../escape", "add goals"} {
		if _, _, err := create(dir, name, time.Now()); err == nil {
			t.Errorf("name %q accepted", name)
		}
	}
	if _, _, err := create(filepath.Join(dir, "missing"), "ok_name", time.Now()); err == nil {
		t.Error("missing directory accepted")
	}
}

func TestRunValidatesArguments(t *testing.T) {
	var out bytes.Buffer
	for _, args := range [][]string{nil, {"create"}, {"create", "a", "b"}} {
		if err := run(args, &out); err == nil {
			t.Errorf("args %v accepted", args)
		}
	}
}
