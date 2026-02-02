package repositories

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type redisClient struct {
	client *redis.Client
}

func NewRedisClient(host, port string) *redisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
	})
	return &redisClient{client: rdb}
}

func (r *redisClient) IncrBy(ctx context.Context, key string, value int64) error {
	if err := r.client.IncrBy(ctx, key, value).Err(); err != nil {
		return fmt.Errorf("redis incrby failure for key %s: %w", key, err)
	}
	return nil
}

func (r *redisClient) Decr(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Decr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis decr failure for key %s: %w", key, err)
	}
	return val, nil
}

func (r *redisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("redis get failure for key %s: %w", key, err)
	}
	return val, nil
}

func (r *redisClient) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	val, err := r.client.SAdd(ctx, key, members...).Result()
	if err != nil {
		return 0, fmt.Errorf("redis sadd failure for key %s: %w", key, err)
	}
	return val, nil
}
