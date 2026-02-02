package repositories

import (
	"fmt"
	"net/http"
)

type HTTPPageFetcher struct{}

func NewPageFetcher() *HTTPPageFetcher {
	return &HTTPPageFetcher{}
}

func (pf *HTTPPageFetcher) Fetch(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	return resp, nil
}
