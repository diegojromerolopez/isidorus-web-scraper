package config

import (
	"os"
)

type Config struct {
	AWSEndpointURL         string
	AWSRegion              string
	AWSAccessKeyID         string
	AWSSecretAccessKey     string
	InputQueueURL          string
	WriterQueueURL         string
	ImageExplainerQueueURL string
	ImagesBucket           string
}

func LoadConfig() Config {
	return Config{
		AWSEndpointURL:         getEnv("AWS_ENDPOINT_URL", "http://localstack:4566"),
		AWSRegion:              getEnv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:         getEnv("AWS_ACCESS_KEY_ID", "test"),
		AWSSecretAccessKey:     getEnv("AWS_SECRET_ACCESS_KEY", "test"),
		InputQueueURL:          getEnv("INPUT_QUEUE_URL", ""),
		WriterQueueURL:         getEnv("WRITER_QUEUE_URL", ""),
		ImageExplainerQueueURL: getEnv("IMAGE_EXPLAINER_QUEUE_URL", ""),
		ImagesBucket:           getEnv("IMAGES_BUCKET", "isidorus-images"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
