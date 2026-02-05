package domain

type IndexMessage struct {
	URL        string `json:"url"`
	Content    string `json:"content"`
	Summary    string `json:"summary"`
	ScrapingID int    `json:"scraping_id"`
	UserID     int    `json:"user_id"`
}
