package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisClient_IncrBy(t *testing.T) {
	tests := []struct {
		name    string
		mock    func(redismock.ClientMock)
		wantErr bool
	}{
		{
			name: "Success",
			mock: func(m redismock.ClientMock) { m.ExpectIncrBy("key", 5).SetVal(5) },
		},
		{
			name:    "Error",
			mock:    func(m redismock.ClientMock) { m.ExpectIncrBy("key", 5).SetErr(errors.New("redis error")) },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			client := &redisClient{client: db}
			tt.mock(mock)

			err := client.IncrBy(context.TODO(), "key", 5)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRedisClient_Decr(t *testing.T) {
	tests := []struct {
		name    string
		mock    func(redismock.ClientMock)
		expect  int64
		wantErr bool
	}{
		{
			name:   "Success",
			mock:   func(m redismock.ClientMock) { m.ExpectDecr("key").SetVal(9) },
			expect: 9,
		},
		{
			name:    "Error",
			mock:    func(m redismock.ClientMock) { m.ExpectDecr("key").SetErr(errors.New("redis error")) },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			client := &redisClient{client: db}
			tt.mock(mock)

			val, err := client.Decr(context.TODO(), "key")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expect, val)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRedisClient_Get(t *testing.T) {
	tests := []struct {
		name    string
		mock    func(redismock.ClientMock)
		expect  string
		wantErr bool
	}{
		{
			name:   "Success",
			mock:   func(m redismock.ClientMock) { m.ExpectGet("key").SetVal("value") },
			expect: "value",
		},
		{
			name:    "Error",
			mock:    func(m redismock.ClientMock) { m.ExpectGet("key").SetErr(errors.New("redis error")) },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			client := &redisClient{client: db}
			tt.mock(mock)

			val, err := client.Get(context.TODO(), "key")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expect, val)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRedisClient_SAdd(t *testing.T) {
	tests := []struct {
		name    string
		mock    func(redismock.ClientMock)
		expect  int64
		wantErr bool
	}{
		{
			name:   "Success",
			mock:   func(m redismock.ClientMock) { m.ExpectSAdd("key", "member").SetVal(1) },
			expect: 1,
		},
		{
			name:    "Error",
			mock:    func(m redismock.ClientMock) { m.ExpectSAdd("key", "member").SetErr(errors.New("redis error")) },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			client := &redisClient{client: db}
			tt.mock(mock)

			val, err := client.SAdd(context.TODO(), "key", "member")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expect, val)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
