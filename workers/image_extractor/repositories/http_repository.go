package repositories

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPRepository struct {
	client *http.Client
}

func NewHTTPRepository() *HTTPRepository {
	return &HTTPRepository{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *HTTPRepository) DownloadImage(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download image, status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}
