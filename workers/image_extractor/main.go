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
)

func main() {
	log.Println("Image Extractor Worker starting (Go)...")
	cfg := config.LoadConfig()

	if cfg.InputQueueURL == "" || cfg.WriterQueueURL == "" {
		log.Fatal("INPUT_QUEUE_URL and WRITER_QUEUE_URL must be set")
	}

	// 1. AWS Config
	// Use background context for initial setup
	setupCtx := context.Background()
	awsCfg, err := awsConfig.LoadDefaultConfig(setupCtx,
		awsConfig.WithRegion(cfg.AWSRegion),
		awsConfig.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// 2. Dependency Injection
	sqsRepo := repositories.NewSQSRepository(awsCfg)
	s3Repo := repositories.NewS3Repository(awsCfg)
	httpRepo := repositories.NewHTTPRepository()

	extractorService := services.NewExtractorService(
		sqsRepo,
		s3Repo,
		httpRepo,
		cfg.WriterQueueURL,
		cfg.ImageExplainerQueueURL,
		cfg.ImagesBucket,
		cfg.ImageExplainerEnabled,
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
			var imageMsg domain.ImageMessage
			if err := json.Unmarshal([]byte(*msg.Body), &imageMsg); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
			} else {
				// Process image
				if err := extractorService.ProcessMessage(ctx, imageMsg); err != nil {
					log.Printf("Error processing image %s: %v", imageMsg.URL, err)
				}
			}

			// Delete message after processing (or if invalid)
			if err := sqsRepo.DeleteMessage(ctx, cfg.InputQueueURL, *msg.ReceiptHandle); err != nil {
				log.Printf("Error deleting message: %v", err)
			}
		}
	}
}
