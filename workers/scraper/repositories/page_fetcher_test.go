package repositories

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageFetcher_Fetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Hello</body></html>"))
	}))
	defer server.Close()

	fetcher := NewPageFetcher()
	resp, err := fetcher.Fetch(server.URL)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPageFetcher_Fetch_Error(t *testing.T) {
	fetcher := NewPageFetcher()
	_, err := fetcher.Fetch("http://invalid-url-that-should-fail")

	assert.Error(t, err)
}
