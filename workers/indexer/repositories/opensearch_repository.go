package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"workers/indexer/domain"
)

type OpenSearchRepository struct {
	client     *opensearch.Client
	otelClient TelemetryClient
}

func NewOpenSearchRepository(client *opensearch.Client, otelClient TelemetryClient) *OpenSearchRepository {
	return &OpenSearchRepository{client: client, otelClient: otelClient}
}

func (r *OpenSearchRepository) IndexDocument(ctx context.Context, msg domain.IndexMessage) error {
	ctx, span := r.otelClient.StartSpan(ctx, "OpenSearchRepository.IndexDocument",
		WithAttribute("url", msg.URL),
		WithAttribute("scrapingID", msg.ScrapingID),
		WithAttribute("userID", msg.UserID),
	)
	defer span.End()

	document := map[string]interface{}{
		"url":         msg.URL,
		"content":     msg.Content,
		"summary":     msg.Summary,
		"scraping_id": msg.ScrapingID,
		"user_id":     msg.UserID,
		"created_at":  time.Now().Format(time.RFC3339),
	}

	body, err := json.Marshal(document)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	req := opensearchapi.IndexRequest{
		Index:      "scraped_pages",
		DocumentID: "", // Auto-generate ID or use a hash of URL
		Body:       strings.NewReader(string(body)),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("failed to execute index request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("error indexing document: %s", res.String())
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return err
	}

	span.SetStatus("ok", "success")
	return nil
}
