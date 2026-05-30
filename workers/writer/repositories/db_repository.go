package repositories

import (
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"
	"workers/writer/domain"
	"workers/writer/models"
)

type PostgresDBRepository struct {
	db         *gorm.DB
	batchSize  int
	otelClient TelemetryClient
}

func NewDBRepository(db *gorm.DB, batchSize int, otelClient TelemetryClient) *PostgresDBRepository {
	if batchSize <= 0 {
		batchSize = 100 // Default
	}
	return &PostgresDBRepository{
		db:         db,
		batchSize:  batchSize,
		otelClient: otelClient,
	}
}

func (repo *PostgresDBRepository) InsertPageData(ctx context.Context, msg domain.WriterMessage) error {
	ctx, span := repo.otelClient.StartSpan(ctx, "PostgresDBRepository.InsertPageData",
		WithAttribute("url", msg.URL),
		WithAttribute("scrapingID", msg.ScrapingID),
	)
	defer span.End()

	// Insert Scraped Page
	page := models.ScrapedPage{
		URL:        msg.URL,
		ScrapingID: msg.ScrapingID,
	}

	if err := repo.db.WithContext(ctx).Create(&page).Error; err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("failed to insert scraped page for URL %s: %w", msg.URL, err)
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

		if err := repo.db.WithContext(ctx).CreateInBatches(links, repo.batchSize).Error; err != nil {
			log.Printf("Error batch inserting links for page %d: %v", page.ID, err)
		}
	}

	span.SetStatus("ok", "success")
	return nil
}

func (repo *PostgresDBRepository) InsertImageExplanation(ctx context.Context, msg domain.WriterMessage) error {
	ctx, span := repo.otelClient.StartSpan(ctx, "PostgresDBRepository.InsertImageExplanation",
		WithAttribute("pageURL", msg.PageURL),
		WithAttribute("s3Path", msg.S3Path),
		WithAttribute("scrapingID", msg.ScrapingID),
	)
	defer span.End()

	// Find the page_id first based on PageURL AND ScrapingID
	var page models.ScrapedPage
	err := repo.db.WithContext(ctx).
		Where("url = ? AND scraping_id = ?", msg.PageURL, msg.ScrapingID).
		Order("scraped_at DESC").
		First(&page).Error

	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("failed to find page for URL %s and id %d: %w", msg.PageURL, msg.ScrapingID, err)
	}

	// Upsert Logic: Check if image exists by S3Path and PageID
	var existingImage models.PageImage
	err = repo.db.WithContext(ctx).Where("page_id = ? AND s3_path = ?", page.ID, msg.S3Path).First(&existingImage).Error

	if err == nil {
		// Image exists, update it (e.g. adding explanation)
		if msg.Explanation != "" {
			existingImage.Explanation = msg.Explanation
		}
		if msg.URL != "" {
			existingImage.ImageURL = msg.URL // Update URL if provided (e.g. signed URL)
		}
		if err := repo.db.WithContext(ctx).Save(&existingImage).Error; err != nil {
			span.RecordError(err)
			span.SetStatus("error", err.Error())
			return fmt.Errorf("failed to update image explanation for S3Path %s: %w", msg.S3Path, err)
		}
	} else {
		// Image does not exist, create it
		image := models.PageImage{
			ScrapingID:  msg.ScrapingID,
			PageID:      page.ID,
			ImageURL:    msg.URL,
			Explanation: msg.Explanation,
			S3Path:      msg.S3Path,
		}
		if err := repo.db.WithContext(ctx).Create(&image).Error; err != nil {
			span.RecordError(err)
			span.SetStatus("error", err.Error())
			return fmt.Errorf("failed to insert image for S3Path %s: %w", msg.S3Path, err)
		}
	}

	span.SetStatus("ok", "success")
	return nil
}

func (repo *PostgresDBRepository) InsertPageSummary(ctx context.Context, msg domain.WriterMessage) error {
	ctx, span := repo.otelClient.StartSpan(ctx, "PostgresDBRepository.InsertPageSummary",
		WithAttribute("url", msg.URL),
		WithAttribute("scrapingID", msg.ScrapingID),
	)
	defer span.End()

	// Update the page summary using URL and ScrapingID
	result := repo.db.WithContext(ctx).
		Model(&models.ScrapedPage{}).
		Where("url = ? AND scraping_id = ?", msg.URL, msg.ScrapingID).
		Update("summary", msg.Summary)

	if result.Error != nil {
		span.RecordError(result.Error)
		span.SetStatus("error", result.Error.Error())
		return fmt.Errorf("failed to update page summary for URL %s: %w", msg.URL, result.Error)
	}

	if result.RowsAffected == 0 {
		err := fmt.Errorf("no page found to update summary for URL %s (ScrapingID %d) - will retry", msg.URL, msg.ScrapingID)
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return err
	}

	log.Printf("Successfully updated summary for URL %s (ScrapingID %d)", msg.URL, msg.ScrapingID)
	span.SetStatus("ok", "success")
	return nil
}

func (repo *PostgresDBRepository) CompleteScraping(ctx context.Context, scrapingID int) error {
	ctx, span := repo.otelClient.StartSpan(ctx, "PostgresDBRepository.CompleteScraping",
		WithAttribute("scrapingID", scrapingID),
	)
	defer span.End()

	// Job completion is now handled entirely in DynamoDB.
	// We keep this hook for now to satisfy the interface,
	// but it no longer modifies PostgreSQL.
	log.Printf("Postgres hook: Scraping %d marked complete (No DB changes)", scrapingID)
	span.SetStatus("ok", "success")
	return nil
}
