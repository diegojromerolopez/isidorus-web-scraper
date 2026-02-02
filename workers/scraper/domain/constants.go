package domain

const (
	// Redis Key Patterns
	RedisKeyVisited = "scrape:%d:visited"
	RedisKeyPending = "scrape:%d:pending"

	// Message Types
	MsgTypePageData         = "page_data"
	MsgTypeImageExplanation = "image_explanation"
	MsgTypeScrapingComplete = "scraping_complete"
)
