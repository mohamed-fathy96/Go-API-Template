package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type UserCache interface {
	GetByID(ctx context.Context, id int64) ([]byte, error)
	Set(ctx context.Context, id int64, data []byte, ttl time.Duration) error
	Delete(ctx context.Context, id int64) error
}

type userCache struct {
	client *RedisClient
	prefix string
}

func NewUserCache(redisClient *RedisClient) UserCache {
	return &userCache{
		client: redisClient,
		prefix: "user:",
	}
}

func (c *userCache) key(id int64) string {
	return fmt.Sprintf("%s%d", c.prefix, id)
}

func (c *userCache) GetByID(ctx context.Context, id int64) ([]byte, error) {
	data, err := c.client.client.Get(ctx, c.key(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // cache miss
		}
		return nil, err
	}
	return data, nil
}

func (c *userCache) Set(ctx context.Context, id int64, data []byte, ttl time.Duration) error {
	return c.client.client.Set(ctx, c.key(id), data, ttl).Err()
}

func (c *userCache) Delete(ctx context.Context, id int64) error {
	return c.client.client.Del(ctx, c.key(id)).Err()
}
