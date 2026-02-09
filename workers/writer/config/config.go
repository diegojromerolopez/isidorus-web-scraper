package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	InputQueueURL          string
	DatabaseURL            string
	DynamoDBTable          string
	BatchSize              int
	AWSEndpointURLSQS      string
	AWSEndpointURLDynamoDB string
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func Load() (*Config, error) {
	batchSizeStr := os.Getenv("DB_BATCH_SIZE")
	batchSize, _ := strconv.Atoi(batchSizeStr)
	if batchSize <= 0 {
		batchSize = 25
	}

	cfg := &Config{
		InputQueueURL:          os.Getenv("INPUT_QUEUE_URL"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		DynamoDBTable:          os.Getenv("DYNAMODB_TABLE"),
		BatchSize:              batchSize,
		AWSEndpointURLSQS:      getEnv("AWS_ENDPOINT_URL_SQS", os.Getenv("AWS_ENDPOINT_URL")),
		AWSEndpointURLDynamoDB: getEnv("AWS_ENDPOINT_URL_DYNAMODB", os.Getenv("AWS_ENDPOINT_URL")),
	}

	if cfg.AWSEndpointURLSQS == "" {
		cfg.AWSEndpointURLSQS = "http://localstack:4566"
	}
	if cfg.AWSEndpointURLDynamoDB == "" {
		cfg.AWSEndpointURLDynamoDB = "http://localstack:4566"
	}

	if cfg.InputQueueURL == "" {
		return nil, fmt.Errorf("INPUT_QUEUE_URL is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}
