package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/opensearch-project/opensearch-go/v2"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	"shared/telemetry"
	indexerConfig "workers/indexer/config"
	"workers/indexer/repositories"
	"workers/indexer/services"
)

func initTracer(serviceName string) (*trace.TracerProvider, error) {
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithSampler(telemetry.GetSamplerFromEnv()),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

func main() {
	tp, err := initTracer("indexer")
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	cfg := indexerConfig.LoadConfig()

	// Create TelemetryClient
	otelClient := repositories.NewTelemetryClient(tp, "indexer")

	// AWS/SQS Client
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.AWSRegion),
		config.WithBaseEndpoint(cfg.SQSEndpointURL),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AWSAccessKeyID, cfg.AWSSecretKey, "")),
		config.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	sqsClient := sqs.NewFromConfig(awsCfg)

	// OpenSearch Client
	osClient, err := opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			ResponseHeaderTimeout: 30 * time.Second,
		},
		Addresses: []string{cfg.OpenSearchURL},
	})
	if err != nil {
		log.Fatalf("error creating OpenSearch client: %s", err)
	}

	// Setup Repositories and Service
	sqsRepo := repositories.NewSQSRepository(sqsClient, cfg.InputQueueURL, otelClient)
	osRepo := repositories.NewOpenSearchRepository(osClient, otelClient)
	indexerService := services.NewIndexerService(sqsRepo, osRepo, otelClient)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	indexerService.Start(ctx)
}
