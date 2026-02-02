package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	config_aws "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"scraped-worker/config"
	"scraped-worker/domain"
	"scraped-worker/repositories"
	"scraped-worker/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	awsCfg, err := config_aws.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	rawSQSClient := sqs.NewFromConfig(awsCfg)
	sqsClient := repositories.NewSQSClient(rawSQSClient)
	pageFetcher := repositories.NewPageFetcher()
	redisClient := repositories.NewRedisClient(cfg.RedisHost, cfg.RedisPort)

	scraperService := services.NewScraperService(
		services.WithSQSClient(sqsClient),
		services.WithRedisClient(redisClient),
		services.WithPageFetcher(pageFetcher),
		services.WithQueues(cfg.InputQueueURL, cfg.WriterQueueURL, cfg.ImageQueueURL),
	)

	log.Println("Scraper worker started (DDD Refactor with community standards)")

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
				var body domain.ScrapeMessage
				if err := json.Unmarshal([]byte(*msg.Body), &body); err != nil {
					log.Printf("failed to unmarshal message: %v", err)
					continue
				}

				scraperService.ProcessMessage(body)

				err := sqsClient.DeleteMessage(ctx, cfg.InputQueueURL, msg.ReceiptHandle)
				if err != nil {
					log.Printf("failed to delete message, %v", err)
				}
			}
		}
	}
}
