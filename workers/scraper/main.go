package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"scraped-worker/domain"
	"scraped-worker/repositories"
	"scraped-worker/services"
)

func main() {
	inputQueueURL := os.Getenv("INPUT_QUEUE_URL")
	writerQueueURL := os.Getenv("WRITER_QUEUE_URL")
	imageQueueURL := os.Getenv("IMAGE_QUEUE_URL")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	rawSQSClient := sqs.NewFromConfig(cfg)
	sqsClient := repositories.NewSQSClient(rawSQSClient)
	pageFetcher := repositories.NewPageFetcher()
	
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	// Fallback/Default for safe local dev if env not set, 
	// though docker-compose sets them.
	if redisHost == "" {
		redisHost = "localhost"
	}
	if redisPort == "" {
		redisPort = "6379"
	}
	redisClient := repositories.NewRedisClient(redisHost, redisPort)

	scraperService := services.NewScraperService(
		sqsClient,
		redisClient,
		pageFetcher,
		inputQueueURL,
		writerQueueURL,
		imageQueueURL,
	)

	log.Println("Scraper worker started (DDD Refactor)")

	for {
		msgOutput, err := sqsClient.ReceiveMessages(context.TODO(), inputQueueURL)
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

			err := sqsClient.DeleteMessage(context.TODO(), inputQueueURL, msg.ReceiptHandle)
			if err != nil {
				log.Printf("failed to delete message, %v", err)
			}
		}
	}
}
