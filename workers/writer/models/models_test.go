package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableNames(t *testing.T) {
	tests := []struct {
		name     string
		model    interface{ TableName() string }
		expected string
	}{
		{"ScrapedPage", &ScrapedPage{}, "scraped_pages"},
		{"PageLink", &PageLink{}, "page_links"},
		{"PageImage", &PageImage{}, "page_images"},
		{"Scraping", &Scraping{}, "scrapings"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.model.TableName())
		})
	}
}
