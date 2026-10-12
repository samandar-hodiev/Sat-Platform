// Command server runs the SAT Platform REST API used by the web app, the Owner dashboard
// and the Android/iOS apps.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/samandar-hodiev/sat-platform/api/internal/config"
	"github.com/samandar-hodiev/sat-platform/api/internal/health"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/database"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/logger"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/storage"
	"github.com/samandar-hodiev/sat-platform/api/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "server:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.Database.URL, cfg.Database.MaxConns)
	if err != nil {
		return err
	}
	defer db.Close()
	pingDatabase(ctx, log, db)

	if cfg.Storage.Driver != "local" {
		return fmt.Errorf("storage driver %q is not supported", cfg.Storage.Driver)
	}
	store, err := storage.NewLocal(cfg.Storage.LocalDir)
	if err != nil {
		return err
	}

	router, err := server.NewRouter(server.Deps{
		Config:  cfg,
		Log:     log,
		Limiter: ratelimit.NewMemory(),
		DB:      db,
		Storage: store,
		Checks: map[string]health.Checker{
			"database": func(ctx context.Context) error { return database.Ping(ctx, db) },
		},
	})
	if err != nil {
		return err
	}
	return server.Run(ctx, cfg, log, router)
}

// pingDatabase reports the database state at startup without failing: the API starts,
// /ready answers 503 until PostgreSQL is reachable, and recovers on its own.
func pingDatabase(ctx context.Context, log *slog.Logger, db *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := database.Ping(ctx, db); err != nil {
		log.Warn("database is not reachable yet; /ready will report unavailable", slog.String("error", err.Error()))
		return
	}
	log.Info("database connected")
}
