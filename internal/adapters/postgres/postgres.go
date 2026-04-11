package postgres

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("unable to create connection pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("unable to ping database: %v", err)
	}

	stat := pool.Stat()
	fmt.Printf("connected to postgres (pool: %d total, %d idle)\n", stat.TotalConns(), stat.IdleConns())

	return pool
}
