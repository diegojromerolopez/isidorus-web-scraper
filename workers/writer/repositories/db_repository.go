package repositories

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"writer-worker/domain"
)

type DBRepository interface {
	InsertPageData(data domain.WriterMessage) error
	InsertImageExplanation(data domain.WriterMessage) error
	CompleteScraping(scrapingID int) error
}

type PostgresDBRepository struct {
	DB        *sql.DB
	BatchSize int
}

func NewDBRepository(db *sql.DB, batchSize int) DBRepository {
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
	var pageID int
	err := repo.DB.QueryRow(`
		INSERT INTO scraped_pages (url, scraping_id)
		VALUES ($1, $2)
		RETURNING id
	`, msg.URL, msg.ScrapingID).Scan(&pageID)
	if err != nil {
		log.Printf("Error inserting into scraped_pages: %v", err)
		return err
	}

	// Insert Page Terms (Denormalized with scraping_id)
	var termRows [][]interface{}
	for term, freq := range msg.Terms {
		termRows = append(termRows, []interface{}{msg.ScrapingID, pageID, term, freq})
	}
	err = repo.batchInsert("page_terms", []string{"scraping_id", "page_id", "term", "frequency"}, termRows)
	if err != nil {
		log.Printf("Error batch inserting terms: %v", err)
	}

	// Insert Page Links (Denormalized with scraping_id)
	var linkRows [][]interface{}
	for _, link := range msg.Links {
		linkRows = append(linkRows, []interface{}{msg.ScrapingID, pageID, link})
	}
	err = repo.batchInsert("page_links", []string{"scraping_id", "source_page_id", "target_url"}, linkRows)
	if err != nil {
		log.Printf("Error batch inserting links: %v", err)
	}

	return nil
}

func (repo *PostgresDBRepository) batchInsert(table string, columns []string, rows [][]interface{}) error {
	if len(rows) == 0 {
		return nil
	}

	for i := 0; i < len(rows); i += repo.BatchSize {
		end := i + repo.BatchSize
		if end > len(rows) {
			end = len(rows)
		}

		batch := rows[i:end]
		sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES ", table, strings.Join(columns, ", "))
		vals := []interface{}{}

		placeholderCount := 1
		for r, row := range batch {
			if r > 0 {
				sqlStr += ", "
			}
			sqlStr += "("
			for c := range row {
				if c > 0 {
					sqlStr += ", "
				}
				sqlStr += fmt.Sprintf("$%d", placeholderCount)
				vals = append(vals, row[c])
				placeholderCount++
			}
			sqlStr += ")"
		}

		_, err := repo.DB.Exec(sqlStr, vals...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (repo *PostgresDBRepository) InsertImageExplanation(msg domain.WriterMessage) error {
	// Need to find the page_id first based on PageURL AND ScrapingID (Int)
	var pageID int
	err := repo.DB.QueryRow(`
		SELECT id FROM scraped_pages WHERE url = $1 AND scraping_id = $2 ORDER BY scraped_at DESC LIMIT 1
	`, msg.PageURL, msg.ScrapingID).Scan(&pageID)

	if err != nil {
		log.Printf("Error finding page for image insertion (url=%s, scraping_id=%d): %v", msg.PageURL, msg.ScrapingID, err)
		return err
	}

	_, err = repo.DB.Exec(`
		INSERT INTO page_images (scraping_id, page_id, image_url, explanation, s3_path)
		VALUES ($1, $2, $3, $4, $5)
	`, msg.ScrapingID, pageID, msg.URL, msg.Explanation, msg.S3Path)

	return err
}

func (repo *PostgresDBRepository) CompleteScraping(scrapingID int) error {
	_, err := repo.DB.Exec(`
		UPDATE scrapings
		SET status = 'COMPLETED', completed_at = NOW()
		WHERE id = $1
	`, scrapingID)
	return err
}
