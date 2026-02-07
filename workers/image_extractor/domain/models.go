package domain

// ImageMessage represents a task for the image extractor
type ImageMessage struct {
	URL         string `json:"url"`
	OriginalURL string `json:"original_url"`
	ScrapingID  int    `json:"scraping_id"`
}

// WriterMessage represents data sent to the writer worker
type WriterMessage struct {
	Type        string `json:"type"` // "image_explanation"
	URL         string `json:"url"`
	OriginalURL string `json:"original_url"`
	PageURL     string `json:"page_url"`
	ScrapingID  int    `json:"scraping_id"`
	S3Path      string `json:"s3_path,omitempty"`
}

// ImageExtractorMessage is synonymous with ImageMessage for now
type ImageExtractorMessage struct {
	ImageURL    string `json:"image_url"`
	OriginalURL string `json:"original_url"`
	ScrapingID  int    `json:"scraping_id"`
	S3Path      string `json:"s3_path"`
}
