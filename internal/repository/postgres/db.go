package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgxpool.Pool and provides a shared connection pool for
// all PostgreSQL repository implementations.
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB creates a connection pool using the provided connection string and
// returns a DB handle. The caller is responsible for calling Close when the
// pool is no longer needed.
func NewDB(ctx context.Context, connString string) (*DB, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("postgres: create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close releases all connections held by the pool.
func (db *DB) Close() {
	db.Pool.Close()
}
