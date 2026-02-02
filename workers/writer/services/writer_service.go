package services

import (
	"fmt"
	"writer-worker/domain"
)

// Consumer-side interface
type DBRepository interface {
	InsertPageData(data domain.WriterMessage) error
	InsertImageExplanation(data domain.WriterMessage) error
	CompleteScraping(scrapingID int) error
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
