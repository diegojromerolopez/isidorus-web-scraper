package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisClient_IncrBy(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &redisClient{client: db}
	ctx := context.TODO()

	// Success
	mock.ExpectIncrBy("key", 5).SetVal(5)
	err := client.IncrBy(ctx, "key", 5)
	assert.NoError(t, err)

	// Error
	mock.ExpectIncrBy("key", 5).SetErr(errors.New("redis error"))
	err = client.IncrBy(ctx, "key", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis incrby failure")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRedisClient_Decr(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &redisClient{client: db}
	ctx := context.TODO()

	// Success
	mock.ExpectDecr("key").SetVal(9)
	val, err := client.Decr(ctx, "key")
	assert.NoError(t, err)
	assert.Equal(t, int64(9), val)

	// Error
	mock.ExpectDecr("key").SetErr(errors.New("redis error"))
	val, err = client.Decr(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis decr failure")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRedisClient_Get(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &redisClient{client: db}
	ctx := context.TODO()

	// Success
	mock.ExpectGet("key").SetVal("value")
	val, err := client.Get(ctx, "key")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)

	// Error
	mock.ExpectGet("key").SetErr(errors.New("redis error"))
	val, err = client.Get(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis get failure")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRedisClient_SAdd(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &redisClient{client: db}
	ctx := context.TODO()

	// Success
	mock.ExpectSAdd("key", "member").SetVal(1)
	val, err := client.SAdd(ctx, "key", "member")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	// Error
	mock.ExpectSAdd("key", "member").SetErr(errors.New("redis error"))
	val, err = client.SAdd(ctx, "key", "member")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis sadd failure")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
