package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"
	"writer-worker/domain"
)

// Consumer-side interface
type DBRepository interface {
	InsertPageData(data domain.WriterMessage) error
	InsertImageExplanation(data domain.WriterMessage) error
	InsertPageSummary(data domain.WriterMessage) error
	CompleteScraping(scrapingID int) error
}

type JobStatusRepository interface {
	UpdateJobStatus(ctx context.Context, jobID string, status string) error
	UpdateJobStatusFull(ctx context.Context, jobID string, status string, completedAt string) error
}

type WriterService struct {
	dbRepo     DBRepository
	statusRepo JobStatusRepository
}

// Functional Options Pattern
type WriterOption func(*WriterService)

func WithDBRepository(r DBRepository) WriterOption {
	return func(s *WriterService) { s.dbRepo = r }
}

func WithJobStatusRepository(r JobStatusRepository) WriterOption {
	return func(s *WriterService) { s.statusRepo = r }
}

func NewWriterService(opts ...WriterOption) *WriterService {
	s := &WriterService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *WriterService) ProcessMessage(msg domain.WriterMessage) error {
	var err error
	ctx := context.Background()

	if msg.Type == domain.MsgTypePageData {
		err = s.dbRepo.InsertPageData(msg)
	} else if msg.Type == domain.MsgTypeImageExplanation {
		err = s.dbRepo.InsertImageExplanation(msg)
	} else if msg.Type == domain.MsgTypePageSummary {
		err = s.dbRepo.InsertPageSummary(msg)
	} else if msg.Type == domain.MsgTypeScrapingComplete {
		// 1. Optional Postgres hook (currently no-op/logging)
		_ = s.dbRepo.CompleteScraping(msg.ScrapingID)

		// 2. Sync to DynamoDB if repository is available - THIS IS THE SOURCE OF TRUTH
		if s.statusRepo != nil {
			jobID := strconv.Itoa(msg.ScrapingID)
			completedAt := time.Now().UTC().Format(time.RFC3339)
			if dErr := s.statusRepo.UpdateJobStatusFull(ctx, jobID, domain.StatusCompleted, completedAt); dErr != nil {
				log.Printf("Error syncing status to DynamoDB for job %s: %v", jobID, dErr)
			}
		}
	} else {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to process message type %s: %w", msg.Type, err)
	}
	return nil
}
