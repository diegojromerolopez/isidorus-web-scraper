package domain

// ScrapeMessage represents the task to scrape a URL
type ScrapeMessage struct {
	URL        string `json:"url"`
	Depth      int    `json:"depth"`
	ScrapingID int    `json:"scraping_id"`
}

// WriterMessage represents data sent to the writer worker
type WriterMessage struct {
	Type        string         `json:"type"` // "page_data" or "job_complete"
	URL         string         `json:"url,omitempty"`
	Terms       map[string]int `json:"terms,omitempty"`
	Links       []string       `json:"links,omitempty"`
	Explanation string         `json:"explanation,omitempty"`
	ScrapingID  int            `json:"scraping_id,omitempty"`
}

// ImageMessage represents a task for the image extractor
type ImageMessage struct {
	URL         string `json:"url"`
	OriginalURL string `json:"original_url"`
	ScrapingID  int    `json:"scraping_id"`
}
