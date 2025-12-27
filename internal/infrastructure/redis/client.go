package redis

import (
	"context"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/config"
	"github.com/redis/go-redis/v9"
)

type Client interface {
	Ping(ctx context.Context) error
	Close() error
	GetRDB() *redis.Client
}

type redisClient struct {
	rdb *redis.Client
}

func NewClient(cfg config.RedisConfig) Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Address(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &redisClient{rdb: rdb}
}

func (c *redisClient) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *redisClient) Close() error {
	return c.rdb.Close()
}

func (c *redisClient) GetRDB() *redis.Client {
	return c.rdb
}
