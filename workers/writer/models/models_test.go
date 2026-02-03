package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableNames(t *testing.T) {
	assert.Equal(t, "scraped_pages", (&ScrapedPage{}).TableName())
	assert.Equal(t, "page_terms", (&PageTerm{}).TableName())
	assert.Equal(t, "page_links", (&PageLink{}).TableName())
	assert.Equal(t, "page_images", (&PageImage{}).TableName())
	assert.Equal(t, "scrapings", (&Scraping{}).TableName())
}
