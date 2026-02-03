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
	err := repo.db.
		Model(&models.Scraping{}).
		Where("id = ?", scrapingID).
		Updates(map[string]interface{}{
			"status":       domain.StatusCompleted,
			"completed_at": gorm.Expr("NOW()"),
		}).Error
	if err != nil {
		return fmt.Errorf("failed to complete scraping for ID %d: %w", scrapingID, err)
	}
	return nil
}

func (repo *PostgresDBRepository) BatchInsertPageData(msgs []domain.WriterMessage) error {
	if len(msgs) == 0 {
		return nil
	}

	// 1. Prepare Pages
	var pages []models.ScrapedPage
	// Map to track correspondence if needed, but array index aligned
	for _, msg := range msgs {
		pages = append(pages, models.ScrapedPage{
			URL:        msg.URL,
			ScrapingID: msg.ScrapingID,
		})
	}

	// 2. Batch Insert Pages (GORM populates IDs in 'pages' slice)
	if err := repo.db.CreateInBatches(&pages, repo.batchSize).Error; err != nil {
		return fmt.Errorf("failed to batch insert pages: %w", err)
	}

	// 3. Prepare Terms and Links
	var allTerms []models.PageTerm
	var allLinks []models.PageLink

	for i, page := range pages {
		// page.ID is now set
		msg := msgs[i]

		if len(msg.Terms) > 0 {
			for term, freq := range msg.Terms {
				allTerms = append(allTerms, models.PageTerm{
					ScrapingID: msg.ScrapingID,
					PageID:     page.ID, // Link to the newly created page
					Term:       term,
					Frequency:  freq,
				})
			}
		}

		if len(msg.Links) > 0 {
			for _, link := range msg.Links {
				allLinks = append(allLinks, models.PageLink{
					ScrapingID:   msg.ScrapingID,
					SourcePageID: page.ID, // Link to the newly created page
					TargetURL:    link,
				})
			}
		}
	}

	// 4. Batch Insert Terms
	if len(allTerms) > 0 {
		if err := repo.db.CreateInBatches(allTerms, repo.batchSize).Error; err != nil {
			log.Printf("Error batch inserting terms: %v", err)
			// Non-fatal? Ideally we verify/retry, but for now log error.
		}
	}

	// 5. Batch Insert Links
	if len(allLinks) > 0 {
		if err := repo.db.CreateInBatches(allLinks, repo.batchSize).Error; err != nil {
			log.Printf("Error batch inserting links: %v", err)
		}
	}

	return nil
}

func (repo *PostgresDBRepository) BatchInsertImageExplanation(msgs []domain.WriterMessage) error {
	if len(msgs) == 0 {
		return nil
	}

	// 1. Resolve Page IDs
	// We need to find Page ID for each (PageURL, ScrapingID)
	// Strategy: Fetch all candidate pages by URLs, then map in memory.
	var urls []string
	for _, msg := range msgs {
		urls = append(urls, msg.PageURL)
	}

	var foundPages []models.ScrapedPage
	// Potential optimization: Filter by ScrapingID as well if possible, but URLs are usually unique enough per scraping?
	// Actually URL might appear in multiple scrapings.
	// Let's just fetch by URLs and then match carefully.
	if err := repo.db.Where("url IN ?", urls).Find(&foundPages).Error; err != nil {
		return fmt.Errorf("failed to fetch pages for batch images: %w", err)
	}

	// Map: Key(ScrapingID, URL) -> PageID
	pageMap := make(map[string]int)
	for _, p := range foundPages {
		key := fmt.Sprintf("%d:%s", p.ScrapingID, p.URL)
		pageMap[key] = p.ID
	}

	// 2. Prepare Images
	var images []models.PageImage
	
	for _, msg := range msgs {
		key := fmt.Sprintf("%d:%s", msg.ScrapingID, msg.PageURL)
		pageID, exists := pageMap[key]
		if !exists {
			log.Printf("Warning: Page not found for Image insert (URL: %s, ScrapingID: %d), skipping.", msg.PageURL, msg.ScrapingID)
			continue
		}

		images = append(images, models.PageImage{
			ScrapingID:  msg.ScrapingID,
			PageID:      pageID,
			ImageURL:    msg.URL,
			Explanation: msg.Explanation,
			S3Path:      msg.S3Path,
		})
	}

	// 3. Batch Insert
	if len(images) > 0 {
		if err := repo.db.CreateInBatches(images, repo.batchSize).Error; err != nil {
			return fmt.Errorf("failed to batch insert images: %w", err)
		}
	}

	return nil
}

func (repo *PostgresDBRepository) BatchInsertPageSummary(msgs []domain.WriterMessage) error {
    // Bulk updating is tricky. For now, fallback to sequential updates inside a transaction?
    // Or just iterate. Concurrency might handle it better than complex bulk UPDATE logic.
    // Let's just iterate for now as Summaries are lower volume (1 per page).
    for _, msg := range msgs {
        _ = repo.InsertPageSummary(msg)
    }
    return nil
}
