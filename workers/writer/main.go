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

	config_aws "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	"shared/telemetry"
	"workers/writer/config"
	"workers/writer/domain"
	"workers/writer/repositories"
	"workers/writer/services"
)

// Consumer-side interface for SQS
type SQSClient interface {
	ReceiveMessages(ctx context.Context, queueURL string, maxMessages int32, waitTime int32) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error
}

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
	tp, err := initTracer("writer")
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Create TelemetryClient
	otelClient := repositories.NewTelemetryClient(tp, "writer")

	// Connect DB using GORM
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	// Connect AWS
	awsCfg, err := config_aws.LoadDefaultConfig(context.Background(),
		config_aws.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	rawSQSClient := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		if cfg.SQSEndpointURL != "" {
			o.BaseEndpoint = &cfg.SQSEndpointURL
		}
	})
	sqsClient := repositories.NewSQSClient(rawSQSClient, otelClient)
	dbRepo := repositories.NewDBRepository(db, cfg.BatchSize, otelClient)

	rawDynamoClient := dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		if cfg.DynamoDBEndpointURL != "" {
			o.BaseEndpoint = &cfg.DynamoDBEndpointURL
		}
	})
	dynamoClient := repositories.NewDynamoDBClient(rawDynamoClient, cfg.DynamoDBTable, otelClient)

	writerService := services.NewWriterService(
		services.WithDBRepository(dbRepo),
		services.WithJobStatusRepository(dynamoClient),
		services.WithTelemetryClient(otelClient),
	)

	log.Println("Writer worker started (DDD Refactor with community standards)")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful Shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, initiating shutdown...", sig)
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("Writer worker shutting down gracefully...")
			return
		default:
			msgOutput, err := sqsClient.ReceiveMessages(ctx, cfg.InputQueueURL, 10, 5)
			if err != nil {
				log.Printf("failed to receive messages: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			if len(msgOutput.Messages) == 0 {
				continue
			}

			// Process batch of messages
			for _, msg := range msgOutput.Messages {
				msgCtx := ctx
				var traceContextHolder struct {
					TraceContext map[string]string `json:"_trace_context"`
				}
				if err := json.Unmarshal([]byte(*msg.Body), &traceContextHolder); err == nil && traceContextHolder.TraceContext != nil {
					msgCtx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(traceContextHolder.TraceContext))
				}

				var body domain.WriterMessage
				if err := json.Unmarshal([]byte(*msg.Body), &body); err != nil {
					log.Printf("failed to unmarshal: %v", err)
					continue
				}

				if err := writerService.ProcessMessage(msgCtx, body); err != nil {
					log.Printf("Failed to process message: %v", err)
				} else {
					// Delete on success
					if err := sqsClient.DeleteMessage(msgCtx, cfg.InputQueueURL, msg.ReceiptHandle); err != nil {
						log.Printf("failed to delete message: %v", err)
					}
				}
			}
		}
	}
}
