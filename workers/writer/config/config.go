package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	InputQueueURL string
	DatabaseURL   string
	DynamoDBTable string
	BatchSize     int
}

func Load() (*Config, error) {
	batchSizeStr := os.Getenv("DB_BATCH_SIZE")
	batchSize, _ := strconv.Atoi(batchSizeStr)
	if batchSize <= 0 {
		batchSize = 25
	}

	cfg := &Config{
		InputQueueURL: os.Getenv("INPUT_QUEUE_URL"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		DynamoDBTable: os.Getenv("DYNAMODB_TABLE"),
		BatchSize:     batchSize,
	}

	if cfg.InputQueueURL == "" {
		return nil, fmt.Errorf("INPUT_QUEUE_URL is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}
