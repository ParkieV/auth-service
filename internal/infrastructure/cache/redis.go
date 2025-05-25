package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ParkieV/auth-service/internal/config"
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisCache(cfg config.RedisConfig) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
		DB:   cfg.DB,
	})
	return &RedisCache{
		client: client,
		ctx:    context.Background(),
	}
}

func (r *RedisCache) Set(key, value string, ttl time.Duration) error {
	return r.client.Set(r.ctx, key, value, ttl).Err()
}

func (r *RedisCache) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

func (r *RedisCache) Delete(key string) error {
	return r.client.Del(r.ctx, key).Err()
}
