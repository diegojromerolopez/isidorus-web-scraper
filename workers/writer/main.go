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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"writer-worker/config"
	"writer-worker/domain"
	"writer-worker/repositories"
	"writer-worker/services"
)

// Consumer-side interface for SQS
type SQSClient interface {
	ReceiveMessages(ctx context.Context, queueURL string, maxMessages int32, waitTime int32) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Connect DB using GORM
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	// Connect AWS
	awsCfg, err := config_aws.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	rawSQSClient := sqs.NewFromConfig(awsCfg)
	sqsClient := repositories.NewSQSClient(rawSQSClient)
	dbRepo := repositories.NewDBRepository(db, cfg.BatchSize)

	rawDynamoClient := dynamodb.NewFromConfig(awsCfg)
	dynamoClient := repositories.NewDynamoDBClient(rawDynamoClient, cfg.DynamoDBTable)

	writerService := services.NewWriterService(
		services.WithDBRepository(dbRepo),
		services.WithJobStatusRepository(dynamoClient),
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
				var body domain.WriterMessage
				if err := json.Unmarshal([]byte(*msg.Body), &body); err != nil {
					log.Printf("failed to unmarshal: %v", err)
					continue
				}

				if err := writerService.ProcessMessage(body); err != nil {
					log.Printf("Failed to process message: %v", err)
				} else {
					// Delete on success
					if err := sqsClient.DeleteMessage(ctx, cfg.InputQueueURL, msg.ReceiptHandle); err != nil {
						log.Printf("failed to delete message: %v", err)
					}
				}
			}
		}
	}
}
