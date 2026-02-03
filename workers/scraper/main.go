package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	config_aws "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"scraped-worker/config"
	"scraped-worker/domain"
	"scraped-worker/repositories"
	"scraped-worker/services"
)

const (
	NumWorkers     = 20
	MaxBatchSize   = 10
	FlushInterval  = 1 * time.Second
	SQSMaxMessages = 10
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
		services.WithQueues(cfg.InputQueueURL, cfg.WriterQueueURL, cfg.ImageQueueURL, cfg.SummarizerQueueURL),
		services.WithFeatureFlags(cfg.ImageExplainerEnabled, cfg.PageSummarizerEnabled),
	)

	log.Println("Scraper worker started (Concurrent Worker Pool)")
	log.Printf("Workers: %d, Batch Size: %d", NumWorkers, MaxBatchSize)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channels
	jobs := make(chan types.Message, NumWorkers*2)
	deletes := make(chan types.Message, NumWorkers*2)

	// WaitGroup for workers
	var workerWg sync.WaitGroup
	// WaitGroup for all (workers + deleter)
	var wg sync.WaitGroup

	// Start Workers
	for i := 0; i < NumWorkers; i++ {
		workerWg.Add(1)
		wg.Add(1)
		go worker(ctx, &workerWg, &wg, scraperService, jobs, deletes, i)
	}

	// Start Batch Deleter
	wg.Add(1)
	go batchDeleter(ctx, &wg, sqsClient, cfg.InputQueueURL, deletes)

	// Graceful Shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, initiating shutdown...", sig)
		cancel()
	}()

	// Main Producer Loop
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
			// Fetch messages
			msgOutput, err := sqsClient.ReceiveMessages(ctx, cfg.InputQueueURL, SQSMaxMessages)
			if err != nil {
				log.Printf("failed to receive messages: %v", err)
				time.Sleep(5 * time.Second) // Backoff
				continue
			}

			if len(msgOutput.Messages) == 0 {
				continue
			}

			// checking for context cancellation before sending to avoid blocking
			for _, msg := range msgOutput.Messages {
				select {
				case jobs <- msg:
				case <-ctx.Done():
					break loop
				}
			}
		}
	}

	log.Println("Main loop exited, waiting for workers to finish...")
	close(jobs)      // Signal workers to stop
	workerWg.Wait()  // Wait for ALL workers to finish writing to deletes
	close(deletes)   // NOW it is safe to close deletes
	wg.Wait()        // Wait for deleter (and workers, implicitly) to finish
	log.Println("Shutdown complete.")
}

func worker(ctx context.Context, workerWg *sync.WaitGroup, wg *sync.WaitGroup, svc *services.ScraperService, jobs <-chan types.Message, deletes chan<- types.Message, id int) {
	defer workerWg.Done()
	defer wg.Done()
	for {
		select {
		case msg, ok := <-jobs:
			if !ok {
				return
			}
			var body domain.ScrapeMessage
			if err := json.Unmarshal([]byte(*msg.Body), &body); err != nil {
				log.Printf("[Worker %d] failed to unmarshal: %v", id, err)
				// Even if failed, we should probably delete it to avoid poison pill loop?
				// For now, let's assume we delete it to accept the error.
				select {
				case deletes <- msg:
				case <-ctx.Done():
					return
				}
				continue
			}

			svc.ProcessMessage(body)

			// Send for deletion
			select {
			case deletes <- msg:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func batchDeleter(ctx context.Context, wg *sync.WaitGroup, client *repositories.AWSSQSClient, queueURL string, deletes <-chan types.Message) {
	defer wg.Done()
	var batch []types.DeleteMessageBatchRequestEntry
	ticker := time.NewTicker(FlushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) > 0 {
			if err := client.DeleteMessageBatch(context.Background(), queueURL, batch); err != nil {
				log.Printf("Failed to delete batch: %v", err)
			}
			batch = nil // Reset
		}
	}

	for {
		select {
		case msg, ok := <-deletes:
			if !ok {
				flush()
				return
			}
			batch = append(batch, types.DeleteMessageBatchRequestEntry{
				Id:            msg.MessageId,
				ReceiptHandle: msg.ReceiptHandle,
			})
			if len(batch) >= MaxBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			flush()
			return
		}
	}
}
