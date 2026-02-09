package config

import (
	"os"
)

type Config struct {
	BaseEndpointURL string
	AWSRegion       string
	AWSAccessKeyID  string
	AWSSecretKey    string
	InputQueueURL   string
	OpenSearchURL   string
	SQSEndpointURL  string
}

func LoadConfig() Config {
	return Config{
		BaseEndpointURL: getEnv("BASE_ENDPOINT_URL", "http://localstack:4566"),
		AWSRegion:       getEnv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:  getEnv("AWS_ACCESS_KEY_ID", "test"),
		AWSSecretKey:    getEnv("AWS_SECRET_ACCESS_KEY", "test"),
		InputQueueURL:   getEnv("INPUT_QUEUE_URL", ""),
		OpenSearchURL:   getEnv("OPENSEARCH_URL", "http://localhost:9200"),
		SQSEndpointURL:  getEnv("SQS_ENDPOINT_URL", getEnv("BASE_ENDPOINT_URL", "http://localstack:4566")),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
