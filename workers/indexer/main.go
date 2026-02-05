package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/opensearch-project/opensearch-go/v2"

	indexerConfig "indexer-worker/config"
	"indexer-worker/repositories"
	"indexer-worker/services"
)

func main() {
	cfg := indexerConfig.LoadConfig()

	// AWS/SQS Client
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           cfg.AWSEndpointURL,
			SigningRegion: cfg.AWSRegion,
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.AWSRegion),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AWSAccessKeyID, cfg.AWSSecretKey, "")),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	sqsClient := sqs.NewFromConfig(awsCfg)

	// OpenSearch Client
	osClient, err := opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
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
