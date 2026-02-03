package services

import (
	"context"
	"testing"
	"writer-worker/domain"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks
type MockDBRepository struct {
	mock.Mock
}

func (m *MockDBRepository) InsertPageData(data domain.WriterMessage) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockDBRepository) InsertImageExplanation(data domain.WriterMessage) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockDBRepository) InsertPageSummary(data domain.WriterMessage) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockDBRepository) CompleteScraping(scrapingID int) error {
	args := m.Called(scrapingID)
	return args.Error(0)
}

func (m *MockDBRepository) BatchInsertPageData(msgs []domain.WriterMessage) error {
	args := m.Called(msgs)
	return args.Error(0)
}

func (m *MockDBRepository) BatchInsertImageExplanation(msgs []domain.WriterMessage) error {
	args := m.Called(msgs)
	return args.Error(0)
}

func (m *MockDBRepository) BatchInsertPageSummary(msgs []domain.WriterMessage) error {
	args := m.Called(msgs)
	return args.Error(0)
}

type MockSQSClient struct {
	mock.Mock
}

func (m *MockSQSClient) ReceiveMessages(ctx context.Context, queueURL string, maxMessages int32, waitTime int32) (*sqs.ReceiveMessageOutput, error) {
	args := m.Called(ctx, queueURL, maxMessages, waitTime)
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)
}

func (m *MockSQSClient) DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error {
	args := m.Called(ctx, queueURL, receiptHandle)
	return args.Error(0)
}

func (m *MockSQSClient) DeleteMessageBatch(ctx context.Context, queueURL string, entries []types.DeleteMessageBatchRequestEntry) error {
	args := m.Called(ctx, queueURL, entries)
	return args.Error(0)
}

func TestProcessMessage_PageData(t *testing.T) {
	mockRepo := new(MockDBRepository)
	s := NewWriterService(WithDBRepository(mockRepo))

	msg := domain.WriterMessage{
		Type: "page_data",
		URL:  "http://example.com",
	}

	mockRepo.On("InsertPageData", msg).Return(nil)

	err := s.ProcessMessage(msg)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestProcessMessage_ImageExplanation(t *testing.T) {
	mockRepo := new(MockDBRepository)
	s := NewWriterService(WithDBRepository(mockRepo))

	msg := domain.WriterMessage{
		Type:        "image_explanation",
		URL:         "http://img.com/1.jpg",
		Explanation: "A nice picture",
	}

	mockRepo.On("InsertImageExplanation", msg).Return(nil)

	err := s.ProcessMessage(msg)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestProcessMessage_UnknownType(t *testing.T) {
	mockRepo := new(MockDBRepository)
	s := NewWriterService(WithDBRepository(mockRepo))

	msg := domain.WriterMessage{
		Type: "unknown",
	}

	err := s.ProcessMessage(msg)

	assert.NoError(t, err) // Returns nil for unknown
	mockRepo.AssertNotCalled(t, "InsertPageData", mock.Anything)
	mockRepo.AssertNotCalled(t, "InsertImageExplanation", mock.Anything)
}

func TestProcessMessage_RepoError(t *testing.T) {
	mockRepo := new(MockDBRepository)
	s := NewWriterService(WithDBRepository(mockRepo))

	msg := domain.WriterMessage{
		Type: "page_data",
	}

	mockRepo.On("InsertPageData", msg).Return(assert.AnError)

	err := s.ProcessMessage(msg)

	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}
func TestProcessMessage_ScrapingComplete(t *testing.T) {
	mockRepo := new(MockDBRepository)
	s := NewWriterService(WithDBRepository(mockRepo))

	msg := domain.WriterMessage{
		Type:         "scraping_complete",
		ScrapingID:   123,
	}

	mockRepo.On("CompleteScraping", 123).Return(nil)

	err := s.ProcessMessage(msg)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestProcessMessage_PageSummary(t *testing.T) {
	mockRepo := new(MockDBRepository)
	s := NewWriterService(WithDBRepository(mockRepo))

	msg := domain.WriterMessage{
		Type:        "page_summary",
		URL:         "http://example.com/page",
		Summary:     "This is a summary",
		ScrapingID:  123,
	}

	mockRepo.On("InsertPageSummary", msg).Return(nil)

	err := s.ProcessMessage(msg)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestProcessBatch(t *testing.T) {
	mockRepo := new(MockDBRepository)
	s := NewWriterService(WithDBRepository(mockRepo))

	msgs := []domain.WriterMessage{
		{Type: "page_data", URL: "http://example.com/1"},
		{Type: "page_data", URL: "http://example.com/2"},
		{Type: "image_explanation", URL: "http://img.com/1.jpg"},
		{Type: "page_summary", URL: "http://example.com/summary"},
		{Type: "scraping_complete", ScrapingID: 123},
	}

	byType := map[string][]domain.WriterMessage{
		"page_data": {msgs[0], msgs[1]},
		"image_explanation": {msgs[2]},
		"page_summary": {msgs[3]},
		"scraping_complete": {msgs[4]},
	}

	mockRepo.On("BatchInsertPageData", byType["page_data"]).Return(nil)
	mockRepo.On("BatchInsertImageExplanation", byType["image_explanation"]).Return(nil)
	mockRepo.On("BatchInsertPageSummary", byType["page_summary"]).Return(nil)
	mockRepo.On("CompleteScraping", 123).Return(nil)

	err := s.ProcessBatch(msgs)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
