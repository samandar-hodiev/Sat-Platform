// Package config loads and validates the API configuration from environment variables.
//
// Every value comes from the environment. In development and test a `.env` file (repository
// root or api/) is read first; variables already set in the process win. In production no
// file is read — the platform injects the variables. Load reports every problem at once so a
// misconfigured server fails at startup with one readable message.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvTest        Environment = "test"
	EnvProduction  Environment = "production"
)

type Config struct {
	Env       Environment
	Port      int
	Version   string
	WebURL    string
	LogLevel  string
	LogFormat string
	Database  DatabaseConfig
	HTTP      HTTPConfig
	RateLimit RateLimitConfig
	Storage   StorageConfig
}

type DatabaseConfig struct {
	URL      string
	MaxConns int32
}

type HTTPConfig struct {
	CORSAllowedOrigins []string
	TrustedProxies     []string
	MaxBodyBytes       int64
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	ShutdownTimeout    time.Duration
}

type RateLimitConfig struct {
	// PerMinute is the default per-client-IP budget for /api/v1. Sensitive endpoints
	// (sign-in, codes, AI) add their own, stricter limits.
	PerMinute int
}

type StorageConfig struct {
	Driver   string
	LocalDir string
}

func (c *Config) IsProduction() bool { return c.Env == EnvProduction }

// Load reads the process environment (plus .env outside production) and validates it.
func Load() (*Config, error) {
	if Environment(os.Getenv("APP_ENV")) != EnvProduction {
		for _, f := range []string{".env", "../.env"} {
			if err := loadDotEnv(f); err != nil {
				return nil, err
			}
		}
	}
	return FromLookup(os.LookupEnv)
}

// FromLookup builds a Config from any key lookup function; tests use it to avoid the
// process environment.
func FromLookup(lookup func(string) (string, bool)) (*Config, error) {
	r := reader{lookup: lookup}
	env := Environment(r.str("APP_ENV", string(EnvDevelopment)))
	defaultFormat := "text"
	if env == EnvProduction {
		defaultFormat = "json"
	}
	webURL := r.str("WEB_URL", "http://localhost:3000")

	cfg := &Config{
		Env:       env,
		Port:      r.int("API_PORT", 8090),
		Version:   r.str("APP_VERSION", "dev"),
		WebURL:    webURL,
		LogLevel:  r.str("LOG_LEVEL", "info"),
		LogFormat: r.str("LOG_FORMAT", defaultFormat),
		Database: DatabaseConfig{
			URL:      r.str("DATABASE_URL", ""),
			MaxConns: int32(r.int("DATABASE_MAX_CONNS", 10)),
		},
		HTTP: HTTPConfig{
			CORSAllowedOrigins: r.list("CORS_ALLOWED_ORIGINS", []string{webURL}),
			TrustedProxies:     r.list("TRUSTED_PROXIES", nil),
			MaxBodyBytes:       int64(r.int("HTTP_MAX_BODY_BYTES", 1<<20)),
			ReadTimeout:        r.duration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:       r.duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout:    r.duration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
		RateLimit: RateLimitConfig{PerMinute: r.int("RATE_LIMIT_PER_MINUTE", 300)},
		Storage: StorageConfig{
			Driver:   r.str("STORAGE_DRIVER", "local"),
			LocalDir: r.str("STORAGE_LOCAL_DIR", "./var/storage"),
		},
	}

	errs := append(r.errs, cfg.validate()...)
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}
	return cfg, nil
}

func (c *Config) validate() []error {
	var errs []error
	switch c.Env {
	case EnvDevelopment, EnvTest, EnvProduction:
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be development, test or production, got %q", c.Env))
	}
	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("API_PORT must be a valid port, got %d", c.Port))
	}
	if c.Database.URL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	} else if u, err := url.Parse(c.Database.URL); err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		// The URL itself is never echoed: it usually contains a password.
		errs = append(errs, errors.New("DATABASE_URL must be a postgres:// URL"))
	}
	if c.Database.MaxConns < 1 {
		errs = append(errs, errors.New("DATABASE_MAX_CONNS must be at least 1"))
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error, got %q", c.LogLevel))
	}
	if c.LogFormat != "text" && c.LogFormat != "json" {
		errs = append(errs, fmt.Errorf("LOG_FORMAT must be text or json, got %q", c.LogFormat))
	}
	if c.HTTP.MaxBodyBytes < 1 {
		errs = append(errs, errors.New("HTTP_MAX_BODY_BYTES must be positive"))
	}
	if c.RateLimit.PerMinute < 1 {
		errs = append(errs, errors.New("RATE_LIMIT_PER_MINUTE must be at least 1"))
	}
	if c.Storage.Driver != "local" {
		errs = append(errs, fmt.Errorf("STORAGE_DRIVER %q is not supported (available: local)", c.Storage.Driver))
	}

	if len(c.HTTP.CORSAllowedOrigins) == 0 {
		errs = append(errs, errors.New("CORS_ALLOWED_ORIGINS must list at least one web origin"))
	}
	for _, origin := range c.HTTP.CORSAllowedOrigins {
		// Browsers are allowed by exact origin only; never a wildcard.
		if strings.Contains(origin, "*") {
			errs = append(errs, fmt.Errorf("CORS_ALLOWED_ORIGINS must list exact origins, got %q", origin))
		} else if c.IsProduction() && !strings.HasPrefix(origin, "https://") {
			errs = append(errs, fmt.Errorf("CORS_ALLOWED_ORIGINS must use https:// in production, got %q", origin))
		}
	}
	if c.IsProduction() {
		if !strings.HasPrefix(c.WebURL, "https://") {
			errs = append(errs, errors.New("WEB_URL must use https:// in production"))
		}
		if c.LogFormat != "json" {
			errs = append(errs, errors.New("LOG_FORMAT must be json in production"))
		}
	}
	return errs
}

// reader collects parse errors so every problem is reported at once.
type reader struct {
	lookup func(string) (string, bool)
	errs   []error
}

func (r *reader) str(key, def string) string {
	if v, ok := r.lookup(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func (r *reader) int(key string, def int) int {
	v := r.str(key, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s must be an integer, got %q", key, v))
		return def
	}
	return n
}

func (r *reader) duration(key string, def time.Duration) time.Duration {
	v := r.str(key, "")
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		r.errs = append(r.errs, fmt.Errorf("%s must be a positive duration like 15s or 2m, got %q", key, v))
		return def
	}
	return d
}

func (r *reader) list(key string, def []string) []string {
	v := r.str(key, "")
	if v == "" {
		return def
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
