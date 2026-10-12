package config

import (
	"errors"
	"net/url"
	"os"
)

// DatabaseURL returns only DATABASE_URL (with the same .env loading as Load). Tools such as
// cmd/migrate use it, so they do not need the API's other settings (JWT secret, origins…).
func DatabaseURL() (string, error) {
	if Environment(os.Getenv("APP_ENV")) != EnvProduction {
		for _, f := range []string{".env", "../.env"} {
			if err := loadDotEnv(f); err != nil {
				return "", err
			}
		}
	}
	return databaseURLFrom(os.LookupEnv)
}

func databaseURLFrom(lookup func(string) (string, bool)) (string, error) {
	raw, _ := lookup("DATABASE_URL")
	if raw == "" {
		return "", errors.New("DATABASE_URL is required")
	}
	if u, err := url.Parse(raw); err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		// The URL itself is never echoed: it usually contains a password.
		return "", errors.New("DATABASE_URL must be a postgres:// URL")
	}
	return raw, nil
}
