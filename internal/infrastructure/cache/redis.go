package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ParkieV/auth-service/internal/config"
)

// RedisCache хранит клиент и контекст
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisCache создаёт и пингует Redis
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

// Set сохраняет ключ с TTL
func (r *RedisCache) Set(key, value string, ttl time.Duration) error {
	return r.client.Set(r.ctx, key, value, ttl).Err()
}

// Get получает значение по ключу
func (r *RedisCache) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

// Delete удаляет ключ
func (r *RedisCache) Delete(key string) error {
	return r.client.Del(r.ctx, key).Err()
}
