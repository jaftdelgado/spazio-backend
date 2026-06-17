package shared

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// Find the .env file relative to this source file's directory
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	envPath := filepath.Join(basepath, "..", "..", ".env")

	_ = godotenv.Load(envPath)

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Fatalf("TEST_DATABASE_URL or DATABASE_URL environment variable is not set in .env")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func WithTransaction(t *testing.T, pool *pgxpool.Pool, fn func(pgx.Tx)) {
	t.Helper()

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			t.Errorf("failed to rollback transaction: %v", err)
		}
	}()

	fn(tx)
}
