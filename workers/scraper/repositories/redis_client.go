package repositories

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisClient interface {
	IncrBy(ctx context.Context, key string, value int64) error
	Decr(ctx context.Context, key string) (int64, error)
	Get(ctx context.Context, key string) (string, error)
	SAdd(ctx context.Context, key string, members ...interface{}) (int64, error)
}

type redisClient struct {
	client *redis.Client
}

func NewRedisClient(host, port string) RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
	})
	return &redisClient{client: rdb}
}

func (r *redisClient) IncrBy(ctx context.Context, key string, value int64) error {
	return r.client.IncrBy(ctx, key, value).Err()
}

func (r *redisClient) Decr(ctx context.Context, key string) (int64, error) {
	return r.client.Decr(ctx, key).Result()
}

func (r *redisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *redisClient) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return r.client.SAdd(ctx, key, members...).Result()
}
