package repositories

import (
	"fmt"
	"log"

	"gorm.io/gorm"
	"writer-worker/domain"
	"writer-worker/models"
)

type PostgresDBRepository struct {
	db        *gorm.DB
	batchSize int
}

func NewDBRepository(db *gorm.DB, batchSize int) *PostgresDBRepository {
	if batchSize <= 0 {
		batchSize = 100 // Default
	}
	return &PostgresDBRepository{
		db:        db,
		batchSize: batchSize,
	}
}

func (repo *PostgresDBRepository) InsertPageData(msg domain.WriterMessage) error {
	// Insert Scraped Page
	page := models.ScrapedPage{
		URL:        msg.URL,
		ScrapingID: msg.ScrapingID,
	}

	if err := repo.db.Create(&page).Error; err != nil {
		return fmt.Errorf("failed to insert scraped page for URL %s: %w", msg.URL, err)
	}

	// Insert Page Terms (Batch)
	if len(msg.Terms) > 0 {
		var terms []models.PageTerm
		for term, freq := range msg.Terms {
			terms = append(terms, models.PageTerm{
				ScrapingID: msg.ScrapingID,
				PageID:     page.ID,
				Term:       term,
				Frequency:  freq,
			})
		}

		if err := repo.db.CreateInBatches(terms, repo.batchSize).Error; err != nil {
			log.Printf("Error batch inserting terms for page %d: %v", page.ID, err)
		}
	}

	// Insert Page Links (Batch)
	if len(msg.Links) > 0 {
		var links []models.PageLink
		for _, link := range msg.Links {
			links = append(links, models.PageLink{
				ScrapingID:   msg.ScrapingID,
				SourcePageID: page.ID,
				TargetURL:    link,
			})
		}

		if err := repo.db.CreateInBatches(links, repo.batchSize).Error; err != nil {
			log.Printf("Error batch inserting links for page %d: %v", page.ID, err)
		}
	}

	return nil
}

func (repo *PostgresDBRepository) InsertImageExplanation(msg domain.WriterMessage) error {
	// Find the page_id first based on PageURL AND ScrapingID
	var page models.ScrapedPage
	err := repo.db.
		Where("url = ? AND scraping_id = ?", msg.PageURL, msg.ScrapingID).
		Order("scraped_at DESC").
		First(&page).Error

	if err != nil {
		return fmt.Errorf("failed to find page for URL %s and id %d: %w", msg.PageURL, msg.ScrapingID, err)
	}

	// Insert the image
	image := models.PageImage{
		ScrapingID:  msg.ScrapingID,
		PageID:      page.ID,
		ImageURL:    msg.URL,
		Explanation: msg.Explanation,
		S3Path:      msg.S3Path,
	}

	if err := repo.db.Create(&image).Error; err != nil {
		return fmt.Errorf("failed to insert image explanation for URL %s: %w", msg.URL, err)
	}
	return nil
}

func (repo *PostgresDBRepository) InsertPageSummary(msg domain.WriterMessage) error {
	// Update the page summary using URL and ScrapingID
	result := repo.db.
		Model(&models.ScrapedPage{}).
		Where("url = ? AND scraping_id = ?", msg.URL, msg.ScrapingID).
		Update("summary", msg.Summary)

	if result.Error != nil {
		return fmt.Errorf("failed to update page summary for URL %s: %w", msg.URL, result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no page found to update summary for URL %s (ScrapingID %d) - will retry", msg.URL, msg.ScrapingID)
	}

	return nil
}

func (repo *PostgresDBRepository) CompleteScraping(scrapingID int) error {
	// Job completion is now handled entirely in DynamoDB.
	// We keep this hook for now to satisfy the interface,
	// but it no longer modifies PostgreSQL.
	log.Printf("Postgres hook: Scraping %d marked complete (No DB changes)", scrapingID)
	return nil
}
