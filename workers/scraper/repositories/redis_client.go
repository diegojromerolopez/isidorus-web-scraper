package repositories

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type redisClient struct {
	client     *redis.Client
	otelClient TelemetryClient
}

func NewRedisClient(host, port string, otelClient TelemetryClient) *redisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
	})
	return &redisClient{client: rdb, otelClient: otelClient}
}

func (r *redisClient) IncrBy(ctx context.Context, key string, value int64) error {
	ctx, span := r.otelClient.StartSpan(ctx, "redisClient.IncrBy", WithAttribute("key", key), WithAttribute("value", value))
	defer span.End()

	if err := r.client.IncrBy(ctx, key, value).Err(); err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("redis incrby failure for key %s: %w", key, err)
	}
	span.SetStatus("ok", "success")
	return nil
}

func (r *redisClient) Decr(ctx context.Context, key string) (int64, error) {
	ctx, span := r.otelClient.StartSpan(ctx, "redisClient.Decr", WithAttribute("key", key))
	defer span.End()

	val, err := r.client.Decr(ctx, key).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return 0, fmt.Errorf("redis decr failure for key %s: %w", key, err)
	}
	span.SetAttribute("result", val)
	span.SetStatus("ok", "success")
	return val, nil
}

func (r *redisClient) Get(ctx context.Context, key string) (string, error) {
	ctx, span := r.otelClient.StartSpan(ctx, "redisClient.Get", WithAttribute("key", key))
	defer span.End()

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return "", fmt.Errorf("redis get failure for key %s: %w", key, err)
	}
	span.SetAttribute("result", val)
	span.SetStatus("ok", "success")
	return val, nil
}

func (r *redisClient) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	ctx, span := r.otelClient.StartSpan(ctx, "redisClient.SAdd", WithAttribute("key", key))
	defer span.End()

	val, err := r.client.SAdd(ctx, key, members...).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return 0, fmt.Errorf("redis sadd failure for key %s: %w", key, err)
	}
	span.SetAttribute("result", val)
	span.SetStatus("ok", "success")
	return val, nil
}
