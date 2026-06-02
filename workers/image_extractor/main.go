package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"workers/image_extractor/config"
	"workers/image_extractor/domain"
	"workers/image_extractor/repositories"
	"workers/image_extractor/services"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"shared/telemetry"
)

func main() {
	log.Println("Image Extractor Worker starting (Go)...")

	tp, err := telemetry.InitTelemetry(context.Background(), "image-extractor")
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	cfg := config.LoadConfig()

	if cfg.InputQueueURL == "" || cfg.WriterQueueURL == "" {
		log.Fatal("INPUT_QUEUE_URL and WRITER_QUEUE_URL must be set")
	}

	// Create TelemetryClient
	otelClient := repositories.NewTelemetryClient(tp, "image-extractor")

	// 1. AWS Config
	// Use background context for initial setup
	setupCtx := context.Background()

	// SQS Config
	sqsAwsCfg, err := awsConfig.LoadDefaultConfig(setupCtx,
		awsConfig.WithRegion(cfg.AWSRegion),
		awsConfig.WithBaseEndpoint(cfg.SQSEndpointURL),
		awsConfig.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	)
	if err != nil {
		log.Fatalf("unable to load SQS SDK config, %v", err)
	}

	// S3 Config
	s3AwsCfg, err := awsConfig.LoadDefaultConfig(setupCtx,
		awsConfig.WithRegion(cfg.AWSRegion),
		awsConfig.WithBaseEndpoint(cfg.S3EndpointURL),
		awsConfig.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	)
	if err != nil {
		log.Fatalf("unable to load S3 SDK config, %v", err)
	}

	// 2. Dependency Injection
	sqsRepo := repositories.NewSQSRepository(sqsAwsCfg, otelClient)
	s3Repo := repositories.NewS3Repository(s3AwsCfg, otelClient)
	httpRepo := repositories.NewHTTPRepository(otelClient)

	extractorService := services.NewExtractorService(
		sqsRepo,
		s3Repo,
		httpRepo,
		cfg.WriterQueueURL,
		cfg.ImageExplainerQueueURL,
		cfg.ImagesBucket,
		cfg.ImageExplainerEnabled,
		otelClient,
	)

	// 3. Main Loop
	// Graceful Shutdown handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, initiating shutdown...", sig)
		cancel()
	}()

	log.Printf("Listening for messages on %s...", cfg.InputQueueURL)
	for {
		select {
		case <-ctx.Done():
			log.Println("Image Extractor shutting down...")
			return
		default:
			// Continue
		}

		messages, err := sqsRepo.ReceiveMessages(ctx, cfg.InputQueueURL)
		if err != nil {
			log.Printf("Error receiving messages: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, msg := range messages {
			msgCtx := ctx
			// Extract trace context from SQS MessageAttributes
			traceContext := make(map[string]string)
			for k, attr := range msg.MessageAttributes {
				if attr.StringValue != nil {
					traceContext[k] = *attr.StringValue
				}
			}
			if len(traceContext) > 0 {
				msgCtx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(traceContext))
			}

			var imageMsg domain.ImageMessage
			if err := json.Unmarshal([]byte(*msg.Body), &imageMsg); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
			} else {
				// Process image
				if err := extractorService.ProcessMessage(msgCtx, imageMsg); err != nil {
					log.Printf("Error processing image %s: %v", imageMsg.URL, err)
				}
			}

			// Delete message after processing (or if invalid)
			if err := sqsRepo.DeleteMessage(msgCtx, cfg.InputQueueURL, *msg.ReceiptHandle); err != nil {
				log.Printf("Error deleting message: %v", err)
			}
		}
	}
}
