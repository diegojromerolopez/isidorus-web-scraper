package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRedisClient(t *testing.T) {
	tests := []struct {
		name string
		host string
		port string
	}{
		{
			name: "Localhost",
			host: "localhost",
			port: "6379",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewRedisClient(tt.host, tt.port)
			assert.NotNil(t, client)
		})
	}
}
