package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

func NewDB(ctx context.Context, dsn string) (*DB, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("connected to postgres successfully")
	return &DB{Pool: pool}, nil
}

// RunMigrations executes initial SQL tables if not existing
func (db *DB) RunMigrations(ctx context.Context) error {
	query := `
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";

	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS urls (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE SET NULL,
		original_url TEXT NOT NULL,
		short_code VARCHAR(32) NOT NULL UNIQUE,
		custom_alias VARCHAR(64) UNIQUE,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		click_count BIGINT NOT NULL DEFAULT 0,
		expires_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);
	CREATE INDEX IF NOT EXISTS idx_urls_custom_alias ON urls(custom_alias);
	CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls(user_id);
	CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls(expires_at);

	CREATE TABLE IF NOT EXISTS clicks (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		url_id UUID NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
		referrer TEXT DEFAULT '',
		user_agent TEXT DEFAULT '',
		ip_address VARCHAR(45) DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_clicks_url_id ON clicks(url_id);
	CREATE INDEX IF NOT EXISTS idx_clicks_created_at ON clicks(created_at);
	`
	_, err := db.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	log.Println("Database schema migrations applied successfully")
	return nil
}
