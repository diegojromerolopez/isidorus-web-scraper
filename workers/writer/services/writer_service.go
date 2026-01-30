package services

import (
	"writer-worker/domain"
	"writer-worker/repositories"
)

type WriterService struct {
	DBRepo repositories.DBRepository
}

func NewWriterService(dbRepo repositories.DBRepository) *WriterService {
	return &WriterService{DBRepo: dbRepo}
}

func (s *WriterService) ProcessMessage(msg domain.WriterMessage) error {
	if msg.Type == "page_data" {
		return s.DBRepo.InsertPageData(msg)
	} else if msg.Type == "image_explanation" {
		return s.DBRepo.InsertImageExplanation(msg)
	} else if msg.Type == "scraping_complete" {
		return s.DBRepo.CompleteScraping(msg.ScrapingID)
	}
	return nil
}
