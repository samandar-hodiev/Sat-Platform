package config

import (
	"strings"
	"testing"
)

func TestDatabaseURLFrom(t *testing.T) {
	lookup := func(v string) func(string) (string, bool) {
		return func(key string) (string, bool) {
			if key == "DATABASE_URL" && v != "" {
				return v, true
			}
			return "", false
		}
	}
	if got, err := databaseURLFrom(lookup("postgres://u@localhost/db")); err != nil || got != "postgres://u@localhost/db" {
		t.Fatalf("valid url: %q, %v", got, err)
	}
	if _, err := databaseURLFrom(lookup("")); err == nil {
		t.Fatal("missing url accepted")
	}
	_, err := databaseURLFrom(lookup("mysql://u:s3cret@host/db"))
	if err == nil || strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("wrong scheme: %v", err)
	}
}
