package services

import (
	"fmt"
	"writer-worker/domain"
)

// Consumer-side interface
type DBRepository interface {
	InsertPageData(data domain.WriterMessage) error
	InsertImageExplanation(data domain.WriterMessage) error
	InsertPageSummary(data domain.WriterMessage) error
	CompleteScraping(scrapingID int) error
	BatchInsertPageData(msgs []domain.WriterMessage) error
	BatchInsertImageExplanation(msgs []domain.WriterMessage) error
	BatchInsertPageSummary(msgs []domain.WriterMessage) error
}

type WriterService struct {
	dbRepo DBRepository
}

// Functional Options Pattern
type WriterOption func(*WriterService)

func WithDBRepository(r DBRepository) WriterOption {
	return func(s *WriterService) { s.dbRepo = r }
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
	if msg.Type == domain.MsgTypePageData {
		err = s.dbRepo.InsertPageData(msg)
	} else if msg.Type == domain.MsgTypeImageExplanation {
		err = s.dbRepo.InsertImageExplanation(msg)
	} else if msg.Type == domain.MsgTypePageSummary {
		err = s.dbRepo.InsertPageSummary(msg)
	} else if msg.Type == domain.MsgTypeScrapingComplete {
		err = s.dbRepo.CompleteScraping(msg.ScrapingID)
	} else {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to process message type %s: %w", msg.Type, err)
	}
	return nil
}

func (s *WriterService) ProcessBatch(msgs []domain.WriterMessage) error {
	byType := make(map[string][]domain.WriterMessage)
	for _, msg := range msgs {
		byType[msg.Type] = append(byType[msg.Type], msg)
	}

	if batch, ok := byType[domain.MsgTypePageData]; ok {
		if err := s.dbRepo.BatchInsertPageData(batch); err != nil {
			return fmt.Errorf("failed to batch insert page data: %w", err)
		}
	}

	if batch, ok := byType[domain.MsgTypeImageExplanation]; ok {
		if err := s.dbRepo.BatchInsertImageExplanation(batch); err != nil {
			return fmt.Errorf("failed to batch insert images: %w", err)
		}
	}

	if batch, ok := byType[domain.MsgTypePageSummary]; ok {
		if err := s.dbRepo.BatchInsertPageSummary(batch); err != nil {
			return fmt.Errorf("failed to batch insert summaries: %w", err)
		}
	}

	if batch, ok := byType[domain.MsgTypeScrapingComplete]; ok {
		for _, msg := range batch {
			if err := s.dbRepo.CompleteScraping(msg.ScrapingID); err != nil {
				return fmt.Errorf("failed to complete scraping %d: %w", msg.ScrapingID, err)
			}
		}
	}

	return nil
}
