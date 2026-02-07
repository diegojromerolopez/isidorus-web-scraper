package repositories

import (
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
			Timeout: 10 * time.Second,
		},
	}
}

func (r *HTTPRepository) DownloadImage(url string) ([]byte, string, error) {
	resp, err := r.client.Get(url)
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
