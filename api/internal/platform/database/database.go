// Package database owns the PostgreSQL connection pool shared by every module.
//
// Modules receive the *pgxpool.Pool and write their own repositories with parameterised
// queries ($1, $2, …). Schema changes are migrations only (api/migrations, see docs/DB.md).
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open creates the pool. It does not require the database to be up: pgx connects lazily,
// so the API can start, report itself not ready on /ready, and recover once PostgreSQL
// is reachable.
func Open(ctx context.Context, url string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		// pgx errors can include the URL; return a message without it.
		return nil, fmt.Errorf("parse DATABASE_URL: invalid connection string")
	}
	cfg.MaxConns = maxConns
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	return pool, nil
}

// Ping checks that PostgreSQL answers within the context deadline.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}
