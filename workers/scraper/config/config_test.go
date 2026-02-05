package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	os.Setenv("INPUT_QUEUE_URL", "http://input")
	os.Setenv("WRITER_QUEUE_URL", "http://writer")
	os.Setenv("IMAGE_QUEUE_URL", "http://image")
	os.Setenv("INDEXER_QUEUE_URL", "http://indexer")
	defer os.Unsetenv("INPUT_QUEUE_URL")
	defer os.Unsetenv("WRITER_QUEUE_URL")
	defer os.Unsetenv("IMAGE_QUEUE_URL")
	defer os.Unsetenv("INDEXER_QUEUE_URL")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "http://input", cfg.InputQueueURL)
	assert.Equal(t, "http://writer", cfg.WriterQueueURL)
	assert.Equal(t, "http://indexer", cfg.IndexerQueueURL)
}
