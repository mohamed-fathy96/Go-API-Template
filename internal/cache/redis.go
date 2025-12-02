package cache

import (
	"context"
	"github.com/redis/go-redis/v9"
	"kabsa/internal/config"
	"kabsa/internal/logging"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(ctx context.Context, cfg config.RedisConfig, logger logging.Logger) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisClient{client: rdb}, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
