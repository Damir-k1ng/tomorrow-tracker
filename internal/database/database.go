// Package database creates and configures the PostgreSQL connection pool.
//
// Repositories depend only on *pgxpool.Pool. The startup sequence here —
// parse URL, open pool, ping, migrate — is the single place where a
// misconfigured database fails the deploy loudly and early.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New parses the DATABASE_URL, opens a pgx connection pool, verifies
// connectivity with a bounded ping, and applies the schema migrations.
//
// A non-nil error here is fatal: the caller should log it and exit, since the
// bot cannot function without its database.
func New(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	// Conservative pool sizing — a polling Telegram bot handles one update at
	// a time, so a small pool is plenty and keeps us well under Railway's
	// PostgreSQL connection limits.
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	// Bounded ping so a wrong host/credentials fails fast instead of hanging
	// the Railway deploy.
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	return pool, nil
}
