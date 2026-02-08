package repositories

import (
	"context"
	"fmt"
	"net/http"
)

type HTTPPageFetcher struct{}

func NewPageFetcher() *HTTPPageFetcher {
	return &HTTPPageFetcher{}
}

func (pf *HTTPPageFetcher) Fetch(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for URL %s: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	return resp, nil
}
