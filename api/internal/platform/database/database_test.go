package database

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOpenRejectsBadURLWithoutEchoingIt(t *testing.T) {
	_, err := Open(context.Background(), "postgres://user:s3cret@%zz/db", 2)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "s3cret") {
		t.Errorf("error leaks the connection string: %v", err)
	}
}

func TestPingFailsWhenDatabaseIsDown(t *testing.T) {
	// Port 1 is never PostgreSQL: Open succeeds (lazy), Ping must fail.
	pool, err := Open(context.Background(), "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := Ping(ctx, pool); err == nil {
		t.Fatal("ping succeeded against a closed port")
	}
}

// TestPostgresIntegration runs only when TEST_DATABASE_URL points at a disposable database.
func TestPostgresIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := Open(context.Background(), url, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := Ping(ctx, pool); err != nil {
		t.Fatalf("ping: %v", err)
	}
	var one int
	if err := pool.QueryRow(ctx, "SELECT $1::int", 1).Scan(&one); err != nil || one != 1 {
		t.Fatalf("query: %v (%d)", err, one)
	}
}
