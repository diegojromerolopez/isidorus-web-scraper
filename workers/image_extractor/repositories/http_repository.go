package repositories

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPRepository struct {
	client     *http.Client
	otelClient TelemetryClient
}

func NewHTTPRepository(otelClient TelemetryClient) *HTTPRepository {
	return &HTTPRepository{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		otelClient: otelClient,
	}
}

func (r *HTTPRepository) DownloadImage(ctx context.Context, url string) ([]byte, string, error) {
	ctx, span := r.otelClient.StartSpan(ctx, "HTTPRepository.DownloadImage",
		WithAttribute("url", url),
	)
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return nil, "", fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("failed to download image, status code: %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return nil, "", err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	span.SetStatus("ok", "success")
	return data, contentType, nil
}
