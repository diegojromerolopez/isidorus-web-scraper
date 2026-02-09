package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/opensearch-project/opensearch-go/v2"

	indexerConfig "workers/indexer/config"
	"workers/indexer/repositories"
	"workers/indexer/services"
)

func main() {
	cfg := indexerConfig.LoadConfig()

	// AWS/SQS Client
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.AWSRegion),
		config.WithBaseEndpoint(cfg.AWSEndpointURLSQS),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AWSAccessKeyID, cfg.AWSSecretKey, "")),
		config.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	sqsClient := sqs.NewFromConfig(awsCfg)

	// OpenSearch Client
	osClient, err := opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			ResponseHeaderTimeout: 30 * time.Second,
		},
		Addresses: []string{cfg.OpenSearchURL},
	})
	if err != nil {
		log.Fatalf("error creating OpenSearch client: %s", err)
	}

	// Setup Repositories and Service
	sqsRepo := repositories.NewSQSRepository(sqsClient, cfg.InputQueueURL)
	osRepo := repositories.NewOpenSearchRepository(osClient)
	indexerService := services.NewIndexerService(sqsRepo, osRepo)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	indexerService.Start(ctx)
}
