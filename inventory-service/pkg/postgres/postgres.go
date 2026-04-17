package postgres

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func NewDB(ctx context.Context, cfg Config) (*DB, error) {
	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	db := &DB{Pool: pool}
	if err := db.ensureSchema(ctx); err != nil {
		return nil, err
	}

	log.Println("Connected to PostgreSQL successfully")
	return db, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

func (db *DB) ensureSchema(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS inventory_auto_inc_ids (
  collection_name TEXT PRIMARY KEY,
  counter BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS products (
  id BIGINT PRIMARY KEY,
  name TEXT NOT NULL,
  category TEXT NOT NULL,
  price DOUBLE PRECISION NOT NULL,
  stock BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);
`
	_, err := db.Pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to ensure postgres schema: %w", err)
	}
	return nil
}
