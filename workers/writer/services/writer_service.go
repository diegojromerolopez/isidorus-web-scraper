package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"
	"workers/writer/domain"
	"workers/writer/repositories"
)

// Consumer-side interface
type DBRepository interface {
	InsertPageData(ctx context.Context, data domain.WriterMessage) error
	InsertImageExplanation(ctx context.Context, data domain.WriterMessage) error
	InsertPageSummary(ctx context.Context, data domain.WriterMessage) error
	CompleteScraping(ctx context.Context, scrapingID int) error
}

type JobStatusRepository interface {
	UpdateJobStatus(ctx context.Context, jobID string, status string) error
	UpdateJobStatusFull(ctx context.Context, jobID string, status string, completedAt string) error
	IncrementLinkCount(ctx context.Context, jobID string, increment int) error
}

type WriterService struct {
	dbRepo     DBRepository
	statusRepo JobStatusRepository
	otelClient repositories.TelemetryClient
}

// Functional Options Pattern
type WriterOption func(*WriterService)

func WithDBRepository(r DBRepository) WriterOption {
	return func(s *WriterService) { s.dbRepo = r }
}

func WithJobStatusRepository(r JobStatusRepository) WriterOption {
	return func(s *WriterService) { s.statusRepo = r }
}

func WithTelemetryClient(c repositories.TelemetryClient) WriterOption {
	return func(s *WriterService) { s.otelClient = c }
}

func NewWriterService(opts ...WriterOption) *WriterService {
	s := &WriterService{}
	for _, opt := range opts {
		opt(s)
	}
	if s.otelClient == nil {
		s.otelClient = repositories.NewNoopTelemetryClient()
	}
	return s
}

func (s *WriterService) ProcessMessage(ctx context.Context, msg domain.WriterMessage) error {
	ctx, span := s.otelClient.StartSpan(ctx, "WriterService.ProcessMessage",
		repositories.WithAttribute("type", msg.Type),
		repositories.WithAttribute("scrapingID", msg.ScrapingID),
		repositories.WithAttribute("url", msg.URL),
	)
	defer span.End()

	var err error

	if msg.Type == domain.MsgTypePageData {
		log.Printf("Writer: Processing PageData for job %d, URL %s", msg.ScrapingID, msg.URL)
		err = s.dbRepo.InsertPageData(ctx, msg)
		if err == nil && s.statusRepo != nil && len(msg.Links) > 0 {
			jobID := strconv.Itoa(msg.ScrapingID)
			log.Printf("Writer: Incrementing links_count for job %s by %d", jobID, len(msg.Links))
			if lErr := s.statusRepo.IncrementLinkCount(ctx, jobID, len(msg.Links)); lErr != nil {
				log.Printf("Error incrementing link count for job %s: %v", jobID, lErr)
			}
		}
	} else if msg.Type == domain.MsgTypeImageExplanation {
		log.Printf("Writer: Processing ImageExplanation for job %d, URL %s", msg.ScrapingID, msg.URL)
		err = s.dbRepo.InsertImageExplanation(ctx, msg)
	} else if msg.Type == domain.MsgTypePageSummary {
		log.Printf("Writer: Processing PageSummary for job %d, URL %s", msg.ScrapingID, msg.URL)
		err = s.dbRepo.InsertPageSummary(ctx, msg)
	} else if msg.Type == domain.MsgTypeScrapingComplete {
		// 1. Optional Postgres hook (currently no-op/logging)
		_ = s.dbRepo.CompleteScraping(ctx, msg.ScrapingID)

		// 2. Sync to DynamoDB if repository is available - THIS IS THE SOURCE OF TRUTH
		if s.statusRepo != nil {
			jobID := strconv.Itoa(msg.ScrapingID)
			completedAt := time.Now().UTC().Format(time.RFC3339)
			if dErr := s.statusRepo.UpdateJobStatusFull(ctx, jobID, domain.StatusCompleted, completedAt); dErr != nil {
				log.Printf("Error syncing status to DynamoDB for job %s: %v", jobID, dErr)
			}
		}
	} else {
		span.SetStatus("ok", "success")
		return nil
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("failed to process message type %s: %w", msg.Type, err)
	}
	span.SetStatus("ok", "success")
	return nil
}
