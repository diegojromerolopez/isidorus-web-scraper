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
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
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
	DeleteMessageBatch(ctx context.Context, queueURL string, entries []types.DeleteMessageBatchRequestEntry) error
}

const (
	MaxBufferSize = 50
	FlushInterval = 2 * time.Second
)

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
	writerService := services.NewWriterService(
		services.WithDBRepository(dbRepo),
	)

	log.Println("Writer worker started (Batch Processing Enabled)")

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

	// Buffers
	var msgBuffer []domain.WriterMessage
	var handleBuffer []types.DeleteMessageBatchRequestEntry
	lastFlush := time.Now()

	// Flush function
	flush := func() {
		if len(msgBuffer) == 0 {
			return
		}

		// Process Batch
		if err := writerService.ProcessBatch(msgBuffer); err != nil {
			log.Printf("Error processing batch: %v", err)
			// On error, we drop the buffer (messages will reappear in SQS after visibility timeout)
			// Alternatively, we could retry or handle partials, but fail-fast is safer here.
		} else {
			// Success -> Delete messages from SQS
			// SQS supports max 10 per batch delete
			chunkSize := 10
			for i := 0; i < len(handleBuffer); i += chunkSize {
				end := i + chunkSize
				if end > len(handleBuffer) {
					end = len(handleBuffer)
				}
				if err := sqsClient.DeleteMessageBatch(ctx, cfg.InputQueueURL, handleBuffer[i:end]); err != nil {
					log.Printf("Failed to delete batch of messages: %v", err)
				}
			}
		}

		// Reset buffers
		msgBuffer = nil
		handleBuffer = nil
		lastFlush = time.Now()
	}

	for {
		select {
		case <-ctx.Done():
			flush() // Try to flush remaining items
			log.Println("Writer worker shutting down gracefully...")
			return
		default:
			// Calculate wait time for SQS long polling
			// Ensure we wake up in time for flush interval
			timeSinceFlush := time.Since(lastFlush)
			remainingTime := FlushInterval - timeSinceFlush
			waitTime := int32(2) // Default 2 seconds
			if remainingTime < 2*time.Second && remainingTime > 0 {
				waitTime = int32(remainingTime.Seconds())
				if waitTime < 1 { waitTime = 1 }
			}

			// If flush needed due to time
			if time.Since(lastFlush) >= FlushInterval && len(msgBuffer) > 0 {
				flush()
				continue
			}

			// Fetch messages
			msgOutput, err := sqsClient.ReceiveMessages(ctx, cfg.InputQueueURL, 10, waitTime)
			if err != nil {
				log.Printf("failed to receive messages: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if len(msgOutput.Messages) > 0 {
				for _, msg := range msgOutput.Messages {
					var body domain.WriterMessage
					if err := json.Unmarshal([]byte(*msg.Body), &body); err != nil {
						log.Printf("failed to unmarshal: %v", err)
						// Delete bad message to avoid loop?
						// sqsClient.DeleteMessage(ctx, cfg.InputQueueURL, msg.ReceiptHandle)
						continue
					}

					msgBuffer = append(msgBuffer, body)
					handleBuffer = append(handleBuffer, types.DeleteMessageBatchRequestEntry{
						Id:            msg.MessageId,
						ReceiptHandle: msg.ReceiptHandle,
					})

					// If we see a completion signal, we should flush the buffer immediately after processing
					// this batch to ensure the completion signal is processed AFTER all page data in this batch.
					// Actually, to be safer, we could even flush BEFORE appending the completion signal,
					// but since they are processed in order in ProcessBatch, appending is fine.
					if body.Type == domain.MsgTypeScrapingComplete {
						flush()
					}
				}

				if len(msgBuffer) >= MaxBufferSize {
					flush()
				}
			}
		}
	}
}
