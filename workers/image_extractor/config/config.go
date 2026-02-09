package config

import (
	"os"
)

type Config struct {
	BaseEndpointURL        string
	AWSRegion              string
	AWSAccessKeyID         string
	AWSSecretAccessKey     string
	InputQueueURL          string
	WriterQueueURL         string
	ImageExplainerQueueURL string
	ImageExplainerEnabled  bool
	ImagesBucket           string
	SQSEndpointURL         string
	S3EndpointURL          string
}

func LoadConfig() Config {
	return Config{
		BaseEndpointURL:        getEnv("BASE_ENDPOINT_URL", "http://localstack:4566"),
		AWSRegion:              getEnv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:         getEnv("AWS_ACCESS_KEY_ID", "test"),
		AWSSecretAccessKey:     getEnv("AWS_SECRET_ACCESS_KEY", "test"),
		InputQueueURL:          getEnv("INPUT_QUEUE_URL", ""),
		WriterQueueURL:         getEnv("WRITER_QUEUE_URL", ""),
		ImageExplainerQueueURL: getEnv("IMAGE_EXPL_QUEUE_URL", ""),
		ImageExplainerEnabled:  getEnv("IMAGE_EXPLAINER_ENABLED", "true") == "true",
		ImagesBucket:           getEnv("IMAGES_BUCKET", "isidorus-images"),
		SQSEndpointURL:         getEnv("SQS_ENDPOINT_URL", getEnv("BASE_ENDPOINT_URL", "http://localstack:4566")),
		S3EndpointURL:          getEnv("S3_ENDPOINT_URL", getEnv("BASE_ENDPOINT_URL", "http://localstack:4566")),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
