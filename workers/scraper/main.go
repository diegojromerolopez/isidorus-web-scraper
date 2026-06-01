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
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	"shared/telemetry"
	"workers/scraper/config"
	"workers/scraper/domain"
	"workers/scraper/repositories"
	"workers/scraper/services"
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
	tp, err := initTracer("scraper")
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("failed to shutdown trace provider: %v", err)
		}
	}()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	awsCfg, err := config_aws.LoadDefaultConfig(context.Background(),
		config_aws.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	otelClient := repositories.NewTelemetryClient(tp, "scraper")

	rawSQSClient := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		if cfg.SQSEndpointURL != "" {
			o.BaseEndpoint = &cfg.SQSEndpointURL
		}
	})
	sqsClient := repositories.NewSQSClient(rawSQSClient, otelClient)
	pageFetcher := repositories.NewPageFetcher(otelClient)
	redisClient := repositories.NewRedisClient(cfg.RedisHost, cfg.RedisPort, otelClient)

	scraperService := services.NewScraperService(
		services.WithSQSClient(sqsClient),
		services.WithRedisClient(redisClient),
		services.WithPageFetcher(pageFetcher),
		services.WithTelemetryClient(otelClient),
		services.WithQueues(cfg.InputQueueURL, cfg.WriterQueueURL, cfg.ImageQueueURL, cfg.SummarizerQueueURL, cfg.IndexerQueueURL),
		services.WithFeatureFlags(cfg.ImageExtractorEnabled, cfg.ImageExplainerEnabled, cfg.PageSummarizerEnabled),
	)

	log.Println("Scraper worker started (DDD Refactor with community standards)")
	log.Printf("Raw Env: IMAGE_EXTRACTOR_ENABLED='%s', IMAGE_EXPLAINER_ENABLED='%s', PAGE_SUMMARIZER_ENABLED='%s'", os.Getenv("IMAGE_EXTRACTOR_ENABLED"), os.Getenv("IMAGE_EXPLAINER_ENABLED"), os.Getenv("PAGE_SUMMARIZER_ENABLED"))
	log.Printf("Feature Flags: ImageExtractorEnabled=%v, ImageExplainerEnabled=%v, PageSummarizerEnabled=%v", cfg.ImageExtractorEnabled, cfg.ImageExplainerEnabled, cfg.PageSummarizerEnabled)

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
			log.Println("Scrapper worker shutting down gracefully...")
			return
		default:
			msgOutput, err := sqsClient.ReceiveMessages(ctx, cfg.InputQueueURL)
			if err != nil {
				log.Printf("failed to receive message, %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			if len(msgOutput.Messages) == 0 {
				continue
			}

			// Process Messages
			for _, msg := range msgOutput.Messages {
				msgCtx := ctx
				var traceContextHolder struct {
					TraceContext map[string]string `json:"_trace_context"`
				}
				if err := json.Unmarshal([]byte(*msg.Body), &traceContextHolder); err == nil && traceContextHolder.TraceContext != nil {
					msgCtx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(traceContextHolder.TraceContext))
				}

				var body domain.ScrapeMessage
				if err := json.Unmarshal([]byte(*msg.Body), &body); err != nil {
					log.Printf("failed to unmarshal message: %v", err)
					continue
				}

				scraperService.ProcessMessage(msgCtx, body)

				err := sqsClient.DeleteMessage(msgCtx, cfg.InputQueueURL, msg.ReceiptHandle)
				if err != nil {
					log.Printf("failed to delete message, %v", err)
				}
			}
		}
	}
}
