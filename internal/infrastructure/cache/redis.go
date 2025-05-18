package cache

import (
	"context"
	"time"

	"github.com/ParkieV/auth-service/internal/config"
	"github.com/redis/go-redis"
)

// RedisCache реализует интерфейс Cache
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisCache создаёт клиент Redis
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

// Set записывает key=value с TTL
func (r *RedisCache) Set(key, value string, ttl time.Duration) error {
	return r.client.Set(r.ctx, key, value, ttl).Err()
}

// Get возвращает value по key
func (r *RedisCache) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

// Delete удаляет key
func (r *RedisCache) Delete(key string) error {
	return r.client.Del(r.ctx, key).Err()
}
