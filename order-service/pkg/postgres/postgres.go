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
CREATE TABLE IF NOT EXISTS order_auto_inc_ids (
  collection_name TEXT PRIMARY KEY,
  counter BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
  id BIGINT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  total_amount DOUBLE PRECISION NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS order_items (
  order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  product_id BIGINT NOT NULL,
  quantity BIGINT NOT NULL,
  total_price DOUBLE PRECISION NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
`
	_, err := db.Pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to ensure postgres schema: %w", err)
	}
	return nil
}
