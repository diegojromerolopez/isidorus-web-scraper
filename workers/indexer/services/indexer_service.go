package services

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"workers/indexer/domain"
	"workers/indexer/repositories"
)

type SQSRepository interface {
	ReceiveMessages(ctx context.Context) ([]domain.IndexMessage, []string, error)
	DeleteMessage(ctx context.Context, handle string) error
}

type OpenSearchRepository interface {
	IndexDocument(ctx context.Context, msg domain.IndexMessage) error
}

type IndexerService struct {
	sqsRepo        SQSRepository
	openSearchRepo OpenSearchRepository
	retryDelay     time.Duration
	otelClient     repositories.TelemetryClient
}

func NewIndexerService(sqsRepo SQSRepository, openSearchRepo OpenSearchRepository, otelClient repositories.TelemetryClient) *IndexerService {
	if otelClient == nil {
		otelClient = repositories.NewNoopTelemetryClient()
	}
	return &IndexerService{
		sqsRepo:        sqsRepo,
		openSearchRepo: openSearchRepo,
		retryDelay:     5 * time.Second,
		otelClient:     otelClient,
	}
}

func (s *IndexerService) Start(ctx context.Context) {
	log.Println("Indexer Service started")
	for {
		select {
		case <-ctx.Done():
			log.Println("Indexer Service stopping...")
			return
		default:
			messages, handles, err := s.sqsRepo.ReceiveMessages(ctx)
			if err != nil {
				log.Printf("Error receiving messages: %v", err)
				time.Sleep(s.retryDelay)
				continue
			}

			for i, msg := range messages {
				log.Printf("Indexing document for URL: %s", msg.URL)
				func() {
					msgCtx := ctx
					if msg.TraceContext != nil {
						msgCtx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(msg.TraceContext))
					}

					msgCtx, span := s.otelClient.StartSpan(msgCtx, "IndexerService.ProcessMessage",
						repositories.WithAttribute("url", msg.URL),
						repositories.WithAttribute("scrapingID", msg.ScrapingID),
					)
					defer span.End()

					if err := s.openSearchRepo.IndexDocument(msgCtx, msg); err != nil {
						log.Printf("Error indexing document %s: %v", msg.URL, err)
						span.RecordError(err)
						span.SetStatus("error", err.Error())
						return
					}

					if err := s.sqsRepo.DeleteMessage(msgCtx, handles[i]); err != nil {
						log.Printf("Error deleting message %s: %v", handles[i], err)
					}
					span.SetStatus("ok", "success")
				}()
			}
		}
	}
}
