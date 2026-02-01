package repositories

import (
	"fmt"
	"net/http"
)

type PageFetcher interface {
	Fetch(url string) (*http.Response, error)
}

type HTTPPageFetcher struct{}

func NewPageFetcher() PageFetcher {
	return &HTTPPageFetcher{}
}

func (pf *HTTPPageFetcher) Fetch(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %v", url, err)
	}
	return resp, nil
}
