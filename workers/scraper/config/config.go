package config

import (
	"fmt"
	"os"
)

type Config struct {
	InputQueueURL  string
	WriterQueueURL string
	ImageQueueURL  string
	RedisHost      string
	RedisPort      string
}

func Load() (*Config, error) {
	cfg := &Config{
		InputQueueURL:  os.Getenv("INPUT_QUEUE_URL"),
		WriterQueueURL: os.Getenv("WRITER_QUEUE_URL"),
		ImageQueueURL:  os.Getenv("IMAGE_QUEUE_URL"),
		RedisHost:      os.Getenv("REDIS_HOST"),
		RedisPort:      os.Getenv("REDIS_PORT"),
	}

	if cfg.InputQueueURL == "" {
		return nil, fmt.Errorf("INPUT_QUEUE_URL is required")
	}
	if cfg.WriterQueueURL == "" {
		return nil, fmt.Errorf("WRITER_QUEUE_URL is required")
	}
	if cfg.ImageQueueURL == "" {
		return nil, fmt.Errorf("IMAGE_QUEUE_URL is required")
	}

	if cfg.RedisHost == "" {
		cfg.RedisHost = "localhost"
	}
	if cfg.RedisPort == "" {
		cfg.RedisPort = "6379"
	}

	return cfg, nil
}
