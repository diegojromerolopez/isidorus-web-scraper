package repositories

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageFetcher_Fetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>Hello</body></html>"))
	}))
	defer server.Close()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name: "Success",
			url:  server.URL,
		},
		{
			name:    "Invalid URL",
			url:     "http://invalid-url-that-should-fail",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := NewPageFetcher()
			resp, err := fetcher.Fetch(context.Background(), tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
}
