package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newPostgresPool(envKey string) (*pgxpool.Pool, error) {
	databaseURL := os.Getenv(envKey)

	if databaseURL == "" {
		return nil, fmt.Errorf("%s is required", envKey)
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool for %s: %w", envKey, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect using %s: %w", envKey, err)
	}

	log.Printf("connected using %s", envKey)

	return pool, nil
}
