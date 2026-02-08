package services

import (
	"context"
	"testing"
	"workers/writer/domain"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks
type MockDBRepository struct {
	mock.Mock
}

func (m *MockDBRepository) InsertPageData(ctx context.Context, data domain.WriterMessage) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockDBRepository) InsertImageExplanation(ctx context.Context, data domain.WriterMessage) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockDBRepository) InsertPageSummary(ctx context.Context, data domain.WriterMessage) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockDBRepository) CompleteScraping(ctx context.Context, scrapingID int) error {
	args := m.Called(ctx, scrapingID)
	return args.Error(0)
}

type MockJobStatusRepository struct {
	mock.Mock
}

func (m *MockJobStatusRepository) UpdateJobStatus(ctx context.Context, jobID string, status string) error {
	args := m.Called(ctx, jobID, status)
	return args.Error(0)
}

func (m *MockJobStatusRepository) UpdateJobStatusFull(ctx context.Context, jobID string, status string, completedAt string) error {
	args := m.Called(ctx, jobID, status, completedAt)
	return args.Error(0)
}

func (m *MockJobStatusRepository) IncrementLinkCount(ctx context.Context, jobID string, increment int) error {
	args := m.Called(ctx, jobID, increment)
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

	mockRepo.On("InsertPageData", mock.Anything, msg).Return(nil)

	err := s.ProcessMessage(context.Background(), msg)

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

	mockRepo.On("InsertImageExplanation", mock.Anything, msg).Return(nil)

	err := s.ProcessMessage(context.Background(), msg)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestProcessMessage_UnknownType(t *testing.T) {
	mockRepo := new(MockDBRepository)
	s := NewWriterService(WithDBRepository(mockRepo))

	msg := domain.WriterMessage{
		Type: "unknown",
	}

	err := s.ProcessMessage(context.Background(), msg)

	assert.NoError(t, err) // Returns nil for unknown
	mockRepo.AssertNotCalled(t, "InsertPageData", mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "InsertImageExplanation", mock.Anything, mock.Anything)
}

func TestProcessMessage_RepoError(t *testing.T) {
	mockRepo := new(MockDBRepository)
	s := NewWriterService(WithDBRepository(mockRepo))

	msg := domain.WriterMessage{
		Type: "page_data",
	}

	mockRepo.On("InsertPageData", mock.Anything, msg).Return(assert.AnError)

	err := s.ProcessMessage(context.Background(), msg)

	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestProcessMessage_ScrapingComplete(t *testing.T) {
	mockDbRepo := new(MockDBRepository)
	mockStatusRepo := new(MockJobStatusRepository)
	s := NewWriterService(
		WithDBRepository(mockDbRepo),
		WithJobStatusRepository(mockStatusRepo),
	)

	msg := domain.WriterMessage{
		Type:       "scraping_complete",
		ScrapingID: 123,
	}

	mockDbRepo.On("CompleteScraping", mock.Anything, 123).Return(nil)
	mockStatusRepo.On("UpdateJobStatusFull", mock.Anything, "123", domain.StatusCompleted, mock.Anything).Return(nil)

	err := s.ProcessMessage(context.Background(), msg)

	assert.NoError(t, err)
	mockDbRepo.AssertExpectations(t)
	mockStatusRepo.AssertExpectations(t)
}

func TestProcessMessage_PageSummary(t *testing.T) {
	mockRepo := new(MockDBRepository)
	s := NewWriterService(WithDBRepository(mockRepo))

	msg := domain.WriterMessage{
		Type:       "page_summary",
		URL:        "http://example.com/page",
		Summary:    "This is a summary",
		ScrapingID: 123,
	}

	mockRepo.On("InsertPageSummary", mock.Anything, msg).Return(nil)

	err := s.ProcessMessage(context.Background(), msg)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestProcessMessage_DynamoFullError(t *testing.T) {
	mockDbRepo := new(MockDBRepository)
	mockStatusRepo := new(MockJobStatusRepository)
	s := NewWriterService(
		WithDBRepository(mockDbRepo),
		WithJobStatusRepository(mockStatusRepo),
	)

	msg := domain.WriterMessage{
		Type:       "scraping_complete",
		ScrapingID: 123,
	}

	mockDbRepo.On("CompleteScraping", mock.Anything, 123).Return(nil)
	mockStatusRepo.On("UpdateJobStatusFull", mock.Anything, "123", domain.StatusCompleted, mock.Anything).Return(assert.AnError)

	err := s.ProcessMessage(context.Background(), msg)

	assert.NoError(t, err) // We log the error but don't fail the message processing
	mockStatusRepo.AssertExpectations(t)
}

func TestProcessMessage_PageData_IncrementsLinks(t *testing.T) {
	mockDbRepo := new(MockDBRepository)
	mockStatusRepo := new(MockJobStatusRepository)
	s := NewWriterService(
		WithDBRepository(mockDbRepo),
		WithJobStatusRepository(mockStatusRepo),
	)

	msg := domain.WriterMessage{
		Type:       "page_data",
		URL:        "http://example.com",
		ScrapingID: 123,
		Links:      []string{"http://link1.com", "http://link2.com"},
	}

	mockDbRepo.On("InsertPageData", mock.Anything, msg).Return(nil)
	mockStatusRepo.On("IncrementLinkCount", mock.Anything, "123", 2).Return(nil)

	err := s.ProcessMessage(context.Background(), msg)

	assert.NoError(t, err)
	mockDbRepo.AssertExpectations(t)
	mockStatusRepo.AssertExpectations(t)
}
