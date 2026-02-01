package repositories

import (
	"log"

	"gorm.io/gorm"
	"writer-worker/domain"
	"writer-worker/models"
)

type DBRepository interface {
	InsertPageData(data domain.WriterMessage) error
	InsertImageExplanation(data domain.WriterMessage) error
	CompleteScraping(scrapingID int) error
}

type PostgresDBRepository struct {
	DB        *gorm.DB
	BatchSize int
}

func NewDBRepository(db *gorm.DB, batchSize int) DBRepository {
	if batchSize <= 0 {
		batchSize = 100 // Default
	}
	return &PostgresDBRepository{
		DB:        db,
		BatchSize: batchSize,
	}
}

func (repo *PostgresDBRepository) InsertPageData(msg domain.WriterMessage) error {
	// Insert Scraped Page
	page := models.ScrapedPage{
		URL:        msg.URL,
		ScrapingID: msg.ScrapingID,
	}

	if err := repo.DB.Create(&page).Error; err != nil {
		log.Printf("Error inserting into scraped_pages: %v", err)
		return err
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

		if err := repo.DB.CreateInBatches(terms, repo.BatchSize).Error; err != nil {
			log.Printf("Error batch inserting terms: %v", err)
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

		if err := repo.DB.CreateInBatches(links, repo.BatchSize).Error; err != nil {
			log.Printf("Error batch inserting links: %v", err)
		}
	}

	return nil
}

func (repo *PostgresDBRepository) InsertImageExplanation(msg domain.WriterMessage) error {
	// Find the page_id first based on PageURL AND ScrapingID
	var page models.ScrapedPage
	err := repo.DB.
		Where("url = ? AND scraping_id = ?", msg.PageURL, msg.ScrapingID).
		Order("scraped_at DESC").
		First(&page).Error

	if err != nil {
		log.Printf("Error finding page for image insertion (url=%s, scraping_id=%d): %v", msg.PageURL, msg.ScrapingID, err)
		return err
	}

	// Insert the image
	image := models.PageImage{
		ScrapingID:  msg.ScrapingID,
		PageID:      page.ID,
		ImageURL:    msg.URL,
		Explanation: msg.Explanation,
		S3Path:      msg.S3Path,
	}

	return repo.DB.Create(&image).Error
}

func (repo *PostgresDBRepository) CompleteScraping(scrapingID int) error {
	return repo.DB.
		Model(&models.Scraping{}).
		Where("id = ?", scrapingID).
		Updates(map[string]interface{}{
			"status":       "COMPLETED",
			"completed_at": gorm.Expr("NOW()"),
		}).Error
}
