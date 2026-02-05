package services

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"scraped-worker/domain"
	"testing"

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
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	args := m.Called(ctx, key, members)
	return int64(args.Int(0)), args.Error(1)
}

type MockPageFetcher struct {
	mock.Mock
}

func (m *MockPageFetcher) Fetch(url string) (*http.Response, error) {
	args := m.Called(url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestProcessMessage_FullFlow(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)

	// Prepare mocks for Redis interactions
	mockRedis.On("Decr", mock.Anything, mock.Anything).Return(1, nil)

	// SAdd calls for cycle detection
	mockRedis.On("SAdd", mock.Anything, "scrape:123:visited", mock.MatchedBy(func(members []interface{}) bool {
		return members[0] == "http://site1.com"
	})).Return(1, nil)
	mockRedis.On("SAdd", mock.Anything, "scrape:123:visited", mock.MatchedBy(func(members []interface{}) bool {
		return members[0] == "http://site2.com"
	})).Return(1, nil)

	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", "summarizer"),
		WithFeatureFlags(true, true),
	)

	html := `
		<html>
			<body>
				<p>Hello world from isidorus</p>
				<a href="http://site2.com">Link 2</a>
				<img src="http://img.com/a.jpg">
			</body>
		</html>
	`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(html)),
	}

	mockFetcher.On("Fetch", "http://site1.com").Return(resp, nil)

	// Expect SendMessage to Writer
	mockSQS.On("SendMessage", mock.Anything, "writer", mock.MatchedBy(func(msg domain.WriterMessage) bool {
		return msg.URL == "http://site1.com" && msg.Terms["hello"] == 1 && len(msg.Links) == 1 && msg.ScrapingID == 123
	})).Return(nil)

	// Expect SendMessage to Image Queue
	mockSQS.On("SendMessage", mock.Anything, "image", mock.MatchedBy(func(msg domain.ImageMessage) bool {
		return msg.URL == "http://img.com/a.jpg"
	})).Return(nil)

	// Expect SendMessage to Summarizer Queue
	mockSQS.On("SendMessage", mock.Anything, "summarizer", mock.MatchedBy(func(msg domain.PageSummaryMessage) bool {
		return msg.URL == "http://site1.com" && len(msg.Content) > 0 && msg.ScrapingID == 123
	})).Return(nil)

	// Expect Redis IncrBy for 1 link
	mockRedis.On("IncrBy", mock.Anything, "scrape:123:pending", int64(1)).Return(nil)

	// Expect SendMessage for recursion
	mockSQS.On("SendMessage", mock.Anything, "input", mock.MatchedBy(func(msg domain.ScrapeMessage) bool {
		return msg.URL == "http://site2.com" && msg.Depth == 1
	})).Return(nil)

	s.ProcessMessage(domain.ScrapeMessage{
		URL:        "http://site1.com",
		Depth:      2,
		ScrapingID: 123,
	})

	mockFetcher.AssertExpectations(t)
	mockSQS.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

func TestProcessMessage_FetchError(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", "summarizer"),
	)

	mockFetcher.On("Fetch", "http://err.com").Return(nil, assert.AnError)
	// SAdd for seed URL
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://err.com"})

	mockSQS.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcessMessage_Non200(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", "summarizer"),
	)

	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewBufferString("not found")),
	}
	mockFetcher.On("Fetch", "http://404.com").Return(resp, nil)
	// SAdd for seed URL
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://404.com"})

	mockSQS.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcessText_Filtering(t *testing.T) {
	s := &ScraperService{}
	terms := make(map[string]int)
	s.processText("The QUICK brown fox! Is in the garden.", terms)

	assert.Equal(t, 1, terms["quick"])
	assert.Equal(t, 1, terms["brown"])
	assert.Equal(t, 1, terms["fox"])
	assert.Equal(t, 1, terms["garden"])
	assert.NotContains(t, terms, "the")
	assert.NotContains(t, terms, "is")
}

func TestProcessMessage_RedisIncrError(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
	)

	html := `<html><body><a href="http://site2.com">Link</a></body></html>`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(html)),
	}
	mockFetcher.On("Fetch", "http://site1.com").Return(resp, nil)
	// SAdd for seed URL
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool { return m[0] == "http://site1.com" })).Return(1, nil)
	// SAdd for the link found
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool { return m[0] == "http://site2.com" })).Return(1, nil)

	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)
	mockRedis.On("IncrBy", mock.Anything, "scrape:123:pending", int64(1)).Return(assert.AnError)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", Depth: 1, ScrapingID: 123})

	mockSQS.AssertNotCalled(t, "SendMessage", mock.Anything, "input", mock.Anything)
}

func TestProcessMessage_SQSSendError_WithCompensation(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
	)

	html := `<html><body><a href="http://site2.com">Link</a></body></html>`
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(html))}
	mockFetcher.On("Fetch", "http://site1.com").Return(resp, nil)
	// SAdd for seed URL
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool { return m[0] == "http://site1.com" })).Return(1, nil)
	// SAdd for the link found
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool { return m[0] == "http://site2.com" })).Return(1, nil)

	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)
	mockRedis.On("IncrBy", mock.Anything, "scrape:123:pending", int64(1)).Return(nil)
	mockSQS.On("SendMessage", mock.Anything, "input", mock.Anything).Return(assert.AnError)
	mockRedis.On("IncrBy", mock.Anything, "scrape:123:pending", int64(-1)).Return(nil)
	mockRedis.On("Decr", mock.Anything, "scrape:123:pending").Return(0, nil)
	mockSQS.On("SendMessage", mock.Anything, "writer", mock.MatchedBy(func(msg domain.WriterMessage) bool {
		return msg.Type == "scraping_complete"
	})).Return(nil)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", Depth: 1, ScrapingID: 123})

	mockRedis.AssertExpectations(t)
	mockSQS.AssertExpectations(t)
}

func TestProcessMessage_CompensationError(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
	)

	html := `<html><body><a href="http://site2.com">Link</a></body></html>`
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(html))}
	mockFetcher.On("Fetch", "http://site1.com").Return(resp, nil)
	// SAdd for seed URL
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool { return m[0] == "http://site1.com" })).Return(1, nil)
	// SAdd for the link found
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool { return m[0] == "http://site2.com" })).Return(1, nil)

	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)
	mockRedis.On("IncrBy", mock.Anything, "scrape:123:pending", int64(1)).Return(nil)
	mockSQS.On("SendMessage", mock.Anything, "input", mock.Anything).Return(assert.AnError)
	mockRedis.On("IncrBy", mock.Anything, "scrape:123:pending", int64(-1)).Return(assert.AnError)
	mockRedis.On("Decr", mock.Anything, "scrape:123:pending").Return(0, nil)
	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", Depth: 1, ScrapingID: 123})

	mockRedis.AssertExpectations(t)
}

func TestProcessMessage_RedisDecrError(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
	)

	html := `<html><body><p>No links</p></body></html>`
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(html))}
	mockFetcher.On("Fetch", "http://site1.com").Return(resp, nil)
	// SAdd for seed URL
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool { return m[0] == "http://site1.com" })).Return(1, nil)

	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)
	mockRedis.On("Decr", mock.Anything, "scrape:123:pending").Return(0, assert.AnError)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", Depth: 1, ScrapingID: 123})

	mockRedis.AssertExpectations(t)
	mockSQS.AssertNotCalled(t, "SendMessage", mock.Anything, "writer", mock.MatchedBy(func(msg domain.WriterMessage) bool {
		return msg.Type == "scraping_complete"
	}))
}

func TestProcessMessage_DepthZero(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
	)

	html := `<html><body><a href="http://site2.com">Link</a></body></html>`
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(html))}
	mockFetcher.On("Fetch", "http://site1.com").Return(resp, nil)

	// SAdd for seed URL
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool {
		return m[0] == "http://site1.com"
	})).Return(1, nil)

	// Expect SendMessage to Writer
	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)

	// Expect Decr to be called (no links to send, so no IncrBy)
	mockRedis.On("Decr", mock.Anything, "scrape:123:pending").Return(1, nil)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", Depth: 0, ScrapingID: 123})

	// Should NOT send any messages to input queue (depth is 0)
	mockSQS.AssertNotCalled(t, "SendMessage", mock.Anything, "input", mock.Anything)
	// Should NOT call IncrBy (no links to send)
	mockRedis.AssertNotCalled(t, "IncrBy", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcessMessage_AlreadyVisitedURL(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
	)

	html := `<html><body><a href="http://site2.com">Link</a></body></html>`
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(html))}
	mockFetcher.On("Fetch", "http://site1.com").Return(resp, nil)

	// SAdd for seed URL
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool {
		return m[0] == "http://site1.com"
	})).Return(1, nil)

	// SAdd for the link - return 0 (already visited)
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool {
		return m[0] == "http://site2.com"
	})).Return(0, nil)

	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)
	mockRedis.On("Decr", mock.Anything, "scrape:123:pending").Return(1, nil)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", Depth: 1, ScrapingID: 123})

	// Should NOT send message to input queue (URL already visited)
	mockSQS.AssertNotCalled(t, "SendMessage", mock.Anything, "input", mock.Anything)
	// Should NOT call IncrBy (no new links to send)
	mockRedis.AssertNotCalled(t, "IncrBy", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcessMessage_NonHTTPLinks(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
	)

	// HTML with relative and non-HTTP links
	html := `<html><body>
		<a href="/relative">Relative</a>
		<a href="#anchor">Anchor</a>
		<a href="mailto:test@example.com">Email</a>
		<a href="javascript:void(0)">JS</a>
	</body></html>`
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(html))}
	mockFetcher.On("Fetch", "http://site1.com").Return(resp, nil)

	// SAdd for seed URL
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool {
		return m[0] == "http://site1.com"
	})).Return(1, nil)

	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)
	mockRedis.On("Decr", mock.Anything, "scrape:123:pending").Return(1, nil)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", Depth: 1, ScrapingID: 123})

	// Should NOT send any messages to input queue (no HTTP links)
	mockSQS.AssertNotCalled(t, "SendMessage", mock.Anything, "input", mock.Anything)
	// Should NOT call IncrBy (no HTTP links to send)
	mockRedis.AssertNotCalled(t, "IncrBy", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcessMessage_SAddErrorDuringCycleDetection(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
	)

	html := `<html><body><a href="http://site2.com">Link</a></body></html>`
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(html))}
	mockFetcher.On("Fetch", "http://site1.com").Return(resp, nil)

	// SAdd for seed URL
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool {
		return m[0] == "http://site1.com"
	})).Return(1, nil)

	// SAdd for the link - return error (Redis failure during cycle detection)
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.MatchedBy(func(m []interface{}) bool {
		return m[0] == "http://site2.com"
	})).Return(0, assert.AnError)

	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)
	mockRedis.On("Decr", mock.Anything, "scrape:123:pending").Return(1, nil)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", Depth: 1, ScrapingID: 123})

	// Should NOT send message to input queue (SAdd error, link skipped)
	mockSQS.AssertNotCalled(t, "SendMessage", mock.Anything, "input", mock.Anything)
	// Should NOT call IncrBy (link was skipped due to error)
	mockRedis.AssertNotCalled(t, "IncrBy", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcessMessage_SQSSendError_PageData(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
	)

	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("html"))}
	mockFetcher.On("Fetch", mock.Anything).Return(resp, nil)
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)
	mockRedis.On("Decr", mock.Anything, mock.Anything).Return(1, nil)

	// Error sending page data
	mockSQS.On("SendMessage", mock.Anything, "writer", mock.MatchedBy(func(m domain.WriterMessage) bool {
		return m.Type == domain.MsgTypePageData
	})).Return(assert.AnError)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", ScrapingID: 123})
	mockSQS.AssertExpectations(t)
}

func TestProcessMessage_SQSSendError_Summary(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", "summarizer"),
		WithFeatureFlags(false, true), // Summarizer enabled
	)

	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("html"))}
	mockFetcher.On("Fetch", mock.Anything).Return(resp, nil)
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)
	mockRedis.On("Decr", mock.Anything, mock.Anything).Return(1, nil)

	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)
	// Error sending summary
	mockSQS.On("SendMessage", mock.Anything, "summarizer", mock.Anything).Return(assert.AnError)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", ScrapingID: 123})
	mockSQS.AssertExpectations(t)
}

func TestProcessMessage_SQSSendError_Image(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
		WithFeatureFlags(true, false), // Image enabled
	)

	html := `<html><body><img src="http://img.com/a.jpg"></body></html>`
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(html))}
	mockFetcher.On("Fetch", mock.Anything).Return(resp, nil)
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)
	mockRedis.On("Decr", mock.Anything, mock.Anything).Return(1, nil)

	mockSQS.On("SendMessage", mock.Anything, "writer", mock.Anything).Return(nil)
	// Error sending image
	mockSQS.On("SendMessage", mock.Anything, "image", mock.Anything).Return(assert.AnError)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", ScrapingID: 123})
	mockSQS.AssertExpectations(t)
}

func TestProcessMessage_SQSSendError_Completion(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
	)

	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("html"))}
	mockFetcher.On("Fetch", mock.Anything).Return(resp, nil)
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)
	mockRedis.On("Decr", mock.Anything, mock.Anything).Return(0, nil)

	mockSQS.On("SendMessage", mock.Anything, "writer", mock.MatchedBy(func(m domain.WriterMessage) bool {
		return m.Type == domain.MsgTypePageData
	})).Return(nil)
	// Error sending completion
	mockSQS.On("SendMessage", mock.Anything, "writer", mock.MatchedBy(func(m domain.WriterMessage) bool {
		return m.Type == domain.MsgTypeScrapingComplete
	})).Return(assert.AnError)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", ScrapingID: 123})
	mockSQS.AssertExpectations(t)
}
func TestProcessMessage_IgnoreScriptAndStyle(t *testing.T) {
	mockSQS := new(MockSQSClient)
	mockRedis := new(MockRedisClient)
	mockFetcher := new(MockPageFetcher)
	s := NewScraperService(
		WithSQSClient(mockSQS),
		WithRedisClient(mockRedis),
		WithPageFetcher(mockFetcher),
		WithQueues("input", "writer", "image", ""),
	)

	// HTML with script and style
	html := `<html>
		<head>
			<style>
				body { color: red; }
				.hidden { display: none; }
			</style>
			<script>
				var secret = "sensitive_variable";
				console.log("runtime_code");
			</script>
		</head>
		<body>
			<p>Visible content</p>
		</body>
	</html>`
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(html))}
	mockFetcher.On("Fetch", "http://site1.com").Return(resp, nil)

	// SAdd for seed URL
	mockRedis.On("SAdd", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)
	mockRedis.On("Decr", mock.Anything, mock.Anything).Return(1, nil)

	// Expect SendMessage to Writer
	mockSQS.On("SendMessage", mock.Anything, "writer", mock.MatchedBy(func(msg domain.WriterMessage) bool {
		// Verify terms
		if msg.Type != domain.MsgTypePageData {
			return true // Ignore other message types if any
		}
		// Content from script/style should NOT be present
		_, hasRed := msg.Terms["red"]
		_, hasSecret := msg.Terms["sensitive_variable"]
		_, hasRuntime := msg.Terms["runtime_code"]

		// Visible content SHOULD be present
		_, hasVisible := msg.Terms["visible"]
		_, hasContent := msg.Terms["content"]

		return !hasRed && !hasSecret && !hasRuntime && hasVisible && hasContent
	})).Return(nil)

	s.ProcessMessage(domain.ScrapeMessage{URL: "http://site1.com", Depth: 1, ScrapingID: 123})

	mockSQS.AssertExpectations(t)
}
