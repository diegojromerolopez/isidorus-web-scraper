package repositories

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/stretchr/testify/assert"
	"indexer-worker/domain"
)

type mockTransport struct {
	Response *http.Response
	Error    error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.Response, m.Error
}

func TestOpenSearchRepository_IndexDocument(t *testing.T) {
	mockRes := &http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(strings.NewReader(`{"result":"created"}`)),
		Header:     make(http.Header),
	}
	client, _ := opensearch.NewClient(opensearch.Config{
		Transport: &mockTransport{Response: mockRes},
	})

	repo := NewOpenSearchRepository(client)
	msg := domain.IndexMessage{URL: "http://test.com", Content: "test", Summary: "sum", ScrapingID: 1, UserID: 1}
	err := repo.IndexDocument(context.TODO(), msg)

	assert.NoError(t, err)
}

func TestOpenSearchRepository_IndexDocument_Error(t *testing.T) {
	mockRes := &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader(`{"error":"internal error"}`)),
		Header:     make(http.Header),
	}
	client, _ := opensearch.NewClient(opensearch.Config{
		Transport: &mockTransport{Response: mockRes},
	})

	repo := NewOpenSearchRepository(client)
	msg := domain.IndexMessage{URL: "http://test.com", Content: "test", Summary: "sum", ScrapingID: 1, UserID: 1}
	err := repo.IndexDocument(context.TODO(), msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error indexing document")
}
