package services

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"workers/scraper/domain"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks
type MockSQSClient struct {
	mock.Mock
}

func (m *MockSQSClient) ReceiveMessages(ctx context.Context, queueURL string) (*sqs.ReceiveMessageOutput, error) {
	args := m.Called(ctx, queueURL)
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)
}

func (m *MockSQSClient) DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error {
	args := m.Called(ctx, queueURL, receiptHandle)
	return args.Error(0)
}

func (m *MockSQSClient) SendMessage(ctx context.Context, queueURL string, msg interface{}) error {
	args := m.Called(ctx, queueURL, msg)
	return args.Error(0)
}

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) IncrBy(ctx context.Context, key string, value int64) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockRedisClient) Decr(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	args := m.Called(ctx, key, members)
	return args.Get(0).(int64), args.Error(1)
}

type MockPageFetcher struct {
	mock.Mock
}

func (m *MockPageFetcher) Fetch(ctx context.Context, url string) (*http.Response, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestScraperService_ProcessMessage(t *testing.T) {
	type mockFetch struct {
		resp *http.Response
		err  error
	}
	type mockSQS struct {
		queueURL string
		err      error
	}
	type mockRedis struct {
		key      string
		val      int64
		err      error
		isSAdd   bool
		isDecr   bool
		isIncrBy bool
	}

	tests := []struct {
		name           string
		msg            domain.ScrapeMessage
		featureFlags   []ScraperOption
		fetchMock      mockFetch
		sqsMocks       []mockSQS
		redisMocks     []mockRedis
		expectMessages []struct {
			queue string
			msg   interface{}
		}
		expectDecr bool
	}{
		{
			name: "Successful Full Scrape",
			msg:  domain.ScrapeMessage{URL: "http://site1.com", Depth: 1, ScrapingID: 123},
			featureFlags: []ScraperOption{
				WithFeatureFlags(true, true, true),
			},
			fetchMock: mockFetch{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`<html><body><p>isidorus</p><a href="http://site2.com">link</a><img src="http://img.com/a.jpg"></body></html>`)),
				},
			},
			redisMocks: []mockRedis{
				{key: "scrape:123:visited", isSAdd: true, val: 1}, // Seed
				{key: "scrape:123:visited", isSAdd: true, val: 1}, // Link 1
				{key: "scrape:123:pending", isDecr: true, val: 1},
				{key: "scrape:123:pending", isIncrBy: true, val: 1},
			},
			expectMessages: []struct {
				queue string
				msg   interface{}
			}{
				{queue: "writer", msg: mock.MatchedBy(func(m domain.WriterMessage) bool { return m.URL == "http://site1.com" })},
				{queue: "indexer", msg: mock.MatchedBy(func(m domain.IndexMessage) bool { return m.URL == "http://site1.com" })},
				{queue: "summarizer", msg: mock.MatchedBy(func(m domain.PageSummaryMessage) bool { return m.URL == "http://site1.com" })},
				{queue: "image", msg: mock.MatchedBy(func(m domain.ImageMessage) bool { return m.URL == "http://img.com/a.jpg" })},
				{queue: "input", msg: mock.MatchedBy(func(m domain.ScrapeMessage) bool { return m.URL == "http://site2.com" && m.Depth == 0 })},
			},
		},
		{
			name:      "Fetch Error - Signals Completion",
			msg:       domain.ScrapeMessage{URL: "http://err.com", ScrapingID: 123},
			fetchMock: mockFetch{err: assert.AnError},
			redisMocks: []mockRedis{
				{key: "scrape:123:visited", isSAdd: true, val: 1},
				{key: "scrape:123:pending", isDecr: true, val: 0}, // completion signal
			},
			expectMessages: []struct {
				queue string
				msg   interface{}
			}{
				{queue: "writer", msg: mock.MatchedBy(func(m domain.WriterMessage) bool { return m.Type == domain.MsgTypeScrapingComplete })},
			},
		},
		{
			name: "Non-200 Response - Signals Completion",
			msg:  domain.ScrapeMessage{URL: "http://404.com", ScrapingID: 123},
			fetchMock: mockFetch{
				resp: &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBufferString("404"))},
			},
			redisMocks: []mockRedis{
				{key: "scrape:123:visited", isSAdd: true, val: 1},
				{key: "scrape:123:pending", isDecr: true, val: 0},
			},
			expectMessages: []struct {
				queue string
				msg   interface{}
			}{
				{queue: "writer", msg: mock.MatchedBy(func(m domain.WriterMessage) bool { return m.Type == domain.MsgTypeScrapingComplete })},
			},
		},
		{
			name: "Cycle Detection - Already Visited",
			msg:  domain.ScrapeMessage{URL: "http://site1.com", Depth: 1, ScrapingID: 123},
			fetchMock: mockFetch{
				resp: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`<html><body><a href="http://visited.com">link</a></body></html>`))},
			},
			redisMocks: []mockRedis{
				{key: "scrape:123:visited", isSAdd: true, val: 1}, // seed
				{key: "scrape:123:visited", isSAdd: true, val: 0}, // already visited
				{key: "scrape:123:pending", isDecr: true, val: 1},
			},
			expectMessages: []struct {
				queue string
				msg   interface{}
			}{
				{queue: "writer", msg: mock.Anything},
				{queue: "indexer", msg: mock.Anything},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSQS := new(MockSQSClient)
			mockRedis := new(MockRedisClient)
			mockFetcher := new(MockPageFetcher)

			opts := []ScraperOption{
				WithSQSClient(mockSQS),
				WithRedisClient(mockRedis),
				WithPageFetcher(mockFetcher),
				WithQueues("input", "writer", "image", "summarizer", "indexer"),
			}
			opts = append(opts, tt.featureFlags...)

			s := NewScraperService(opts...)

			// Setup Mocks
			mockFetcher.On("Fetch", mock.Anything, tt.msg.URL).Return(tt.fetchMock.resp, tt.fetchMock.err)

			for _, rm := range tt.redisMocks {
				if rm.isSAdd {
					mockRedis.On("SAdd", mock.Anything, rm.key, mock.Anything).Return(rm.val, rm.err).Once()
				}
				if rm.isDecr {
					mockRedis.On("Decr", mock.Anything, rm.key).Return(rm.val, rm.err).Once()
				}
				if rm.isIncrBy {
					mockRedis.On("IncrBy", mock.Anything, rm.key, mock.Anything).Return(rm.err).Once()
				}
			}

			// Capture sent messages
			for _, em := range tt.expectMessages {
				mockSQS.On("SendMessage", mock.Anything, em.queue, em.msg).Return(nil)
			}

			// Handle noise/default behaviors
			mockSQS.On("SendMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

			s.ProcessMessage(context.Background(), tt.msg)

			mockFetcher.AssertExpectations(t)
			mockRedis.AssertExpectations(t)
			mockSQS.AssertExpectations(t)
		})
	}
}


func TestProcessMessage_EmptyQueues(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	// Don't set optional queues
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "", "", ""), // image, summarizer, indexer empty
		WithFeatureFlags(true, true, true),        // Flags enabled but queues empty
	)

	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("<html></html>"))}
	mockFetcher.On("Fetch", mock.Anything, mock.Anything).Return(resp, nil)
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.Anything).Return(int64(1), nil)
	mockRedis.On("Decr", mock.Anything, mock.Anything).Return(int64(1), nil)

	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)

	// Should NOT send to indexer, summarizer, or image because queues are empty strings
	s.ProcessMessage(context.Background(), domain.ScrapeMessage{URL: "http://site1.com", ScrapingID: 123})

	mockSQS.AssertNotCalled(t, "SendMessage", mock.Anything, "indexer", mock.Anything)
	mockSQS.AssertNotCalled(t, "SendMessage", mock.Anything, "summarizer", mock.Anything)
	mockSQS.AssertNotCalled(t, "SendMessage", mock.Anything, "image", mock.Anything)
}
