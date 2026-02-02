package services

import (
	"context"
	"testing"
	"writer-worker/domain"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
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
