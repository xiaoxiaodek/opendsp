package data

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
	"github.com/redis/go-redis/v9"
)

type Data struct {
	Pool    *pgxpool.Pool
	Queries *dbsqlc.Queries
	Rdb     *redis.Client
}

func NewData(ctx context.Context) (*Data, func(), error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://opendsp:opendsp@localhost:5432/opendsp?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse database URL: %w", err)
	}
	config.MaxConns = 20
	config.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, nil, fmt.Errorf("create connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, nil, fmt.Errorf("ping database: %w", err)
	}

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr, DB: 0})
	if err := rdb.Ping(ctx).Err(); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("redis ping: %w", err)
	}

	cleanup := func() {
		pool.Close()
		rdb.Close()
	}

	return &Data{
		Pool:    pool,
		Queries: dbsqlc.New(pool),
		Rdb:     rdb,
	}, cleanup, nil
}
