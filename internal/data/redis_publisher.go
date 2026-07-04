package data

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// RedisPublisher implements biz.EventPublisher backed by Redis PubSub.
type RedisPublisher struct {
	rdb *redis.Client
}

func NewRedisPublisher(rdb *redis.Client) *RedisPublisher {
	return &RedisPublisher{rdb: rdb}
}

func (p *RedisPublisher) Publish(ctx context.Context, channel string, data []byte) error {
	return p.rdb.Publish(ctx, channel, data).Err()
}
