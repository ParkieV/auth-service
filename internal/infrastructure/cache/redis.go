package cache

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ParkieV/auth-service/internal/config"
)

var ErrKeyNotFound = errors.New("key not found")

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	SwapRefresh(ctx context.Context, userID, oldRT, newRT string, ttl time.Duration) (bool, error)
	Delete(ctx context.Context, key string) error
}

type RedisCache struct {
	client *redis.Client
	log    *slog.Logger
}

func NewRedisCache(cfg config.RedisConfig, log *slog.Logger) *RedisCache {
	c := redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
		DB:   cfg.DB,
	})
	return &RedisCache{client: c, log: log}
}

func (r *RedisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrKeyNotFound
	}
	return val, err
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache) SwapRefresh(
	ctx context.Context,
	userID string,
	oldRT string,
	newRT string,
	ttl time.Duration,
) (bool, error) {
	script := redis.NewScript(`
		local oldVal = redis.call("GET", KEYS[1])
		if not oldVal or oldVal ~= ARGV[1] then
			return 0
		end
		redis.call("SET", KEYS[2], ARGV[1], "EX", ARGV[2])
		redis.call("DEL", KEYS[1])
		return 1
	`)

	res, err := script.Run(ctx, r.client, []string{oldRT, newRT}, userID, int(ttl.Seconds())).Result()
	if err != nil {
		r.log.Error("lua SwapRefresh failed", "err", err)
		return false, err
	}

	success, ok := res.(int64)
	if !ok || success != 1 {
		r.log.Warn("SwapRefresh denied", "oldRT", oldRT, "expectedUserID", userID)
		return false, nil
	}

	return true, nil
}
