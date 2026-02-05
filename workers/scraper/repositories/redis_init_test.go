package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRedisClient(t *testing.T) {
	// This just tests the constructor since the methods are already tested
	client := NewRedisClient("localhost", "6379")
	assert.NotNil(t, client)
}
