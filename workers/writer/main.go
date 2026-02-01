package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"writer-worker/domain"
	"writer-worker/repositories"
	"writer-worker/services"
)

func main() {
	inputQueueURL := os.Getenv("INPUT_QUEUE_URL")
	dbURL := os.Getenv("DATABASE_URL")
	batchSizeStr := os.Getenv("DB_BATCH_SIZE")
	batchSize, _ := strconv.Atoi(batchSizeStr)
	if batchSize <= 0 {
		batchSize = 25 // Default for safety
	}

	// Connect DB using GORM
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	// Connect AWS
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	rawSQSClient := sqs.NewFromConfig(cfg)
	sqsClient := repositories.NewSQSClient(rawSQSClient)
	dbRepo := repositories.NewDBRepository(db, batchSize)
	writerService := services.NewWriterService(dbRepo)

	log.Println("Writer worker started (DDD Refactor)")

	for {
		msgOutput, err := sqsClient.ReceiveMessages(context.TODO(), inputQueueURL, 10, 5)
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
			var body domain.WriterMessage
			if err := json.Unmarshal([]byte(*msg.Body), &body); err != nil {
				log.Printf("failed to unmarshal: %v", err)
				continue
			}

			err := writerService.ProcessMessage(body)
			if err != nil {
				log.Printf("Failed to process message: %v", err)
			} else {
				// Delete on success
				err := sqsClient.DeleteMessage(context.TODO(), inputQueueURL, msg.ReceiptHandle)
				if err != nil {
					log.Printf("failed to delete message: %v", err)
				}
			}
		}
	}
}
