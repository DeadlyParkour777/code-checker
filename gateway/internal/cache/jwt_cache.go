package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type JWTCache interface {
	GetUserID(ctx context.Context, token string) (string, error)
	SetUserID(ctx context.Context, token, userID string) error
}

type redisJWTCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisJWTCache(client *redis.Client) JWTCache {
	return &redisJWTCache{
		client: client,
		ttl:    5 * time.Minute,
	}
}

func (c *redisJWTCache) GetUserID(ctx context.Context, token string) (string, error) {
	key := "jwt:" + token
	return c.client.Get(ctx, key).Result()
}

func (c *redisJWTCache) SetUserID(ctx context.Context, token, userID string) error {
	key := "jwt:" + token
	return c.client.Set(ctx, key, userID, c.ttl).Err()
}
