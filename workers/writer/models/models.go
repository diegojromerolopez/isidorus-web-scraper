package models

import (
	"time"
)

// Scraping represents a web scraping job
type Scraping struct {
	ID          int        `gorm:"primaryKey;autoIncrement"`
	URL         string     `gorm:"type:text;not null"`
	Status      string     `gorm:"type:text;not null;default:PENDING"`
	CreatedAt   time.Time  `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP"`
	CompletedAt *time.Time `gorm:"type:timestamp with time zone"`
}

// TableName overrides the table name
func (Scraping) TableName() string {
	return "scrapings"
}

// ScrapedPage represents a scraped web page
type ScrapedPage struct {
	ID         int       `gorm:"primaryKey;autoIncrement"`
	ScrapingID int       `gorm:"not null;index"`
	URL        string    `gorm:"type:text;not null;index:idx_scraped_pages_url"`
	Summary    string    `gorm:"type:text"`
	ScrapedAt  time.Time `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP"`
	
	// Relationships
	Scraping   Scraping    `gorm:"foreignKey:ScrapingID;constraint:OnDelete:CASCADE"`
	Terms      []PageTerm  `gorm:"foreignKey:PageID;constraint:OnDelete:CASCADE"`
	Links      []PageLink  `gorm:"foreignKey:SourcePageID;constraint:OnDelete:CASCADE"`
	Images     []PageImage `gorm:"foreignKey:PageID;constraint:OnDelete:CASCADE"`
}

// TableName overrides the table name
func (ScrapedPage) TableName() string {
	return "scraped_pages"
}

// PageTerm represents a term found on a page
type PageTerm struct {
	ID         int    `gorm:"primaryKey;autoIncrement"`
	ScrapingID int    `gorm:"not null"`
	PageID     int    `gorm:"not null"`
	Term       string `gorm:"type:text;not null;index:idx_page_terms_term"`
	Frequency  int    `gorm:"default:1"`
	
	// Relationships
	Scraping Scraping    `gorm:"foreignKey:ScrapingID;constraint:OnDelete:CASCADE"`
	Page     ScrapedPage `gorm:"foreignKey:PageID;constraint:OnDelete:CASCADE"`
}

// TableName overrides the table name
func (PageTerm) TableName() string {
	return "page_terms"
}

// PageImage represents an image found on a page
type PageImage struct {
	ID          int    `gorm:"primaryKey;autoIncrement"`
	ScrapingID  int    `gorm:"not null"`
	PageID      int    `gorm:"not null"`
	ImageURL    string `gorm:"column:image_url;type:text;not null"`
	Explanation string `gorm:"type:text"`
	S3Path      string `gorm:"column:s3_path;type:text"`
	
	// Relationships
	Scraping Scraping    `gorm:"foreignKey:ScrapingID;constraint:OnDelete:CASCADE"`
	Page     ScrapedPage `gorm:"foreignKey:PageID;constraint:OnDelete:CASCADE"`
}

// TableName overrides the table name
func (PageImage) TableName() string {
	return "page_images"
}

// PageLink represents a link from one page to another
type PageLink struct {
	ID           int    `gorm:"primaryKey;autoIncrement"`
	ScrapingID   int    `gorm:"not null"`
	SourcePageID int    `gorm:"column:source_page_id;not null"`
	TargetURL    string `gorm:"column:target_url;type:text;not null"`
	
	// Relationships
	Scraping   Scraping    `gorm:"foreignKey:ScrapingID;constraint:OnDelete:CASCADE"`
	SourcePage ScrapedPage `gorm:"foreignKey:SourcePageID;constraint:OnDelete:CASCADE"`
}

// TableName overrides the table name
func (PageLink) TableName() string {
	return "page_links"
}
