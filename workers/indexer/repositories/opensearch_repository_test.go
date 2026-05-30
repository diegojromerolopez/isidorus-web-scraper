package repositories

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/stretchr/testify/assert"
	"workers/indexer/domain"
)

type mockTransport struct {
	Response *http.Response
	Error    error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.Response, m.Error
}

func TestOpenSearchRepository_IndexDocument(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "Success",
			statusCode: 201,
			body:       `{"result":"created"}`,
		},
		{
			name:       "Error",
			statusCode: 500,
			body:       `{"error":"internal error"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRes := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
				Header:     make(http.Header),
			}
			client, _ := opensearch.NewClient(opensearch.Config{
				Transport: &mockTransport{Response: mockRes},
			})

			repo := NewOpenSearchRepository(client, NewNoopTelemetryClient())
			msg := domain.IndexMessage{URL: "http://test.com", Content: "test"}
			err := repo.IndexDocument(context.TODO(), msg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
