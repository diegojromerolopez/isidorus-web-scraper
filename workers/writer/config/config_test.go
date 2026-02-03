package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	os.Setenv("INPUT_QUEUE_URL", "http://input")
	os.Setenv("WRITER_QUEUE_URL", "http://writer")
	os.Setenv("DATABASE_URL", "postgres://test")
	defer os.Unsetenv("INPUT_QUEUE_URL")
	defer os.Unsetenv("WRITER_QUEUE_URL")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "http://input", cfg.InputQueueURL)
	assert.Equal(t, "postgres://test", cfg.DatabaseURL)
}
