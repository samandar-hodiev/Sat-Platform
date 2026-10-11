package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func lookupFrom(env map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}
}

func validEnv() map[string]string {
	return map[string]string{"DATABASE_URL": "postgres://localhost:5432/sat_platform?sslmode=disable"}
}

func TestDefaults(t *testing.T) {
	cfg, err := FromLookup(lookupFrom(validEnv()))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != EnvDevelopment || cfg.Port != 8090 || cfg.LogFormat != "text" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if len(cfg.HTTP.CORSAllowedOrigins) != 1 || cfg.HTTP.CORSAllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("CORS origins should default to WEB_URL, got %v", cfg.HTTP.CORSAllowedOrigins)
	}
	if cfg.HTTP.ShutdownTimeout != 15*time.Second {
		t.Errorf("shutdown timeout = %v", cfg.HTTP.ShutdownTimeout)
	}
}

func TestReportsEveryProblemAtOnce(t *testing.T) {
	_, err := FromLookup(lookupFrom(map[string]string{
		"APP_ENV":  "staging",
		"API_PORT": "eighty",
	}))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"APP_ENV", "API_PORT", "DATABASE_URL is required"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s: %v", want, err)
		}
	}
}

func TestDatabaseURLIsNeverEchoed(t *testing.T) {
	env := validEnv()
	env["DATABASE_URL"] = "mysql://user:s3cret@db/x"
	_, err := FromLookup(lookupFrom(env))
	if err == nil || strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("want an error that hides the URL, got %v", err)
	}
}

func TestCORSOriginsMustBeExact(t *testing.T) {
	env := validEnv()
	env["CORS_ALLOWED_ORIGINS"] = "*"
	if _, err := FromLookup(lookupFrom(env)); err == nil || !strings.Contains(err.Error(), "exact origins") {
		t.Errorf("wildcard origin accepted: %v", err)
	}
}

func TestProductionIsStricter(t *testing.T) {
	env := validEnv()
	env["APP_ENV"] = "production"
	env["WEB_URL"] = "http://sat.example"
	env["LOG_FORMAT"] = "text"
	_, err := FromLookup(lookupFrom(env))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"WEB_URL must use https://", "CORS_ALLOWED_ORIGINS must use https://", "LOG_FORMAT must be json"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %q: %v", want, err)
		}
	}

	env = validEnv()
	env["APP_ENV"] = "production"
	env["WEB_URL"] = "https://sat.example"
	cfg, err := FromLookup(lookupFrom(env))
	if err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("production log format should default to json, got %s", cfg.LogFormat)
	}
}

func TestDotEnvDoesNotOverrideProcessEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\nSAT_TEST_A=from-file\nexport SAT_TEST_B=\"quoted value\"\nSAT_TEST_C=plain # trailing comment\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SAT_TEST_A", "from-process")
	for _, k := range []string{"SAT_TEST_B", "SAT_TEST_C"} {
		k := k
		t.Cleanup(func() { os.Unsetenv(k) })
	}

	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("SAT_TEST_A"); got != "from-process" {
		t.Errorf("A = %q, process value must win", got)
	}
	if got := os.Getenv("SAT_TEST_B"); got != "quoted value" {
		t.Errorf("B = %q", got)
	}
	if got := os.Getenv("SAT_TEST_C"); got != "plain" {
		t.Errorf("C = %q", got)
	}
	if err := loadDotEnv(filepath.Join(dir, "missing.env")); err != nil {
		t.Errorf("missing file must be ignored: %v", err)
	}
}
