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
	userID, oldRT, newRT string,
	ttl time.Duration,
) (bool, error) {

	const maxRetry = 3

	for i := 0; i < maxRetry; i++ {
		err := r.client.Watch(ctx, func(tx *redis.Tx) error {
			val, err := tx.Get(ctx, oldRT).Result()
			switch {
			case errors.Is(err, redis.Nil):
				return redis.TxFailedErr
			case err != nil:
				return err
			case val != userID:
				return redis.TxFailedErr
			}

			_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
				if err := p.SetNX(ctx, newRT, userID, ttl).Err(); err != nil {
					return err
				}
				if ok, _ := p.Get(ctx, newRT).Int(); ok == 0 {
					return redis.TxFailedErr
				}
				p.Del(ctx, oldRT)
				return nil
			})
			return err
		}, oldRT, newRT)

		switch {
		case err == nil:
			return true, nil
		case errors.Is(err, redis.TxFailedErr):
			continue
		default:
			return false, err
		}
	}
	return false, nil
}
