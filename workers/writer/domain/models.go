package domain

type WriterMessage struct {
	Type        string         `json:"type"` // "page_data" | "image_explanation" | "scraping_complete"
	URL         string         `json:"url,omitempty"`
	Terms       map[string]int `json:"terms,omitempty"`
	Links       []string       `json:"links,omitempty"`
	Explanation string         `json:"explanation,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	S3Path      string         `json:"s3_path,omitempty"`
	ScrapingID  int            `json:"scraping_id,omitempty"`
	PageURL     string         `json:"page_url,omitempty"` // Link to parent page
}
