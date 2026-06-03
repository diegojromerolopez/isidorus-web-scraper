package repositories

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type HTTPPageFetcher struct {
	otelClient TelemetryClient
}

func NewPageFetcher(otelClient TelemetryClient) *HTTPPageFetcher {
	return &HTTPPageFetcher{otelClient: otelClient}
}

func (pf *HTTPPageFetcher) Fetch(ctx context.Context, url string) (*http.Response, error) {
	ctx, span := pf.otelClient.StartSpan(ctx, "HTTPPageFetcher.Fetch", WithAttribute("url", url))
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return nil, fmt.Errorf("failed to create request for URL %s: %w", url, err)
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	resp, err := client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	span.SetStatus("ok", "success")
	return resp, nil
}
