package domain

// ScrapeMessage represents the task to scrape a URL
type ScrapeMessage struct {
	URL        string `json:"url"`
	Depth      int    `json:"depth"`
	ScrapingID int    `json:"scraping_id"`
	UserID     int    `json:"user_id"`
}

// WriterMessage represents data sent to the writer worker
type WriterMessage struct {
	Type        string   `json:"type"` // "page_data", "job_complete", "page_summary"
	URL         string   `json:"url,omitempty"`
	Links       []string `json:"links,omitempty"`
	Explanation string   `json:"explanation,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	ScrapingID  int      `json:"scraping_id,omitempty"`
}

// ImageMessage represents a task for the image extractor
type ImageMessage struct {
	URL         string `json:"url"`
	OriginalURL string `json:"original_url"`
	ScrapingID  int    `json:"scraping_id"`
}

// PageSummaryMessage represents a task for the page summarizer
type PageSummaryMessage struct {
	URL        string `json:"url"`
	Content    string `json:"content"`
	ScrapingID int    `json:"scraping_id"`
	UserID     int    `json:"user_id"`
}
