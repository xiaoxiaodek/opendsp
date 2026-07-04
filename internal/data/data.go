package data

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opendsp/opendsp/internal/config"
	"github.com/opendsp/opendsp/internal/data/dbsqlc"
	"github.com/redis/go-redis/v9"
)

type Data struct {
	Pool    *pgxpool.Pool
	Queries *dbsqlc.Queries
	Rdb     *redis.Client
}

func NewData(ctx context.Context, dbCfg config.DatabaseConfig, redisCfg config.RedisConfig) (*Data, func(), error) {
	cfg, err := pgxpool.ParseConfig(dbCfg.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse database URL: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, nil, fmt.Errorf("ping database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisCfg.Addr, DB: 0})
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
