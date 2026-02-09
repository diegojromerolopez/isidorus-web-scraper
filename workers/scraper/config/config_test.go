package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name string
		envs map[string]string
	}{
		{
			name: "Basic Config",
			envs: map[string]string{
				"INPUT_QUEUE_URL":   "http://input",
				"WRITER_QUEUE_URL":  "http://writer",
				"IMAGE_QUEUE_URL":   "http://image",
				"INDEXER_QUEUE_URL": "http://indexer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envs {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}
			cfg, err := Load()
			assert.NoError(t, err)
			assert.Equal(t, tt.envs["INPUT_QUEUE_URL"], cfg.InputQueueURL)
			assert.Equal(t, tt.envs["WRITER_QUEUE_URL"], cfg.WriterQueueURL)
		})
	}
}
