package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"image-extractor-worker/config"
	"image-extractor-worker/domain"
	"image-extractor-worker/repositories"
	"image-extractor-worker/services"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
)

func main() {
	log.Println("Image Extractor Worker starting (Go)...")
	cfg := config.LoadConfig()

	if cfg.InputQueueURL == "" || cfg.WriterQueueURL == "" {
		log.Fatal("INPUT_QUEUE_URL and WRITER_QUEUE_URL must be set")
	}

	// 1. AWS Config
	ctx := context.TODO()
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithRegion(cfg.AWSRegion),
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
	)

	// 3. Main Loop
	log.Printf("Listening for messages on %s...", cfg.InputQueueURL)
	for {
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
