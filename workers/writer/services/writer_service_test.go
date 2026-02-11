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

func TestWriterService_ProcessMessage(t *testing.T) {
	tests := []struct {
		name          string
		msg           domain.WriterMessage
		setupMocks    func(*MockDBRepository, *MockJobStatusRepository)
		expectedError error
		expectNoCalls bool
	}{
		{
			name: "Page Data Success",
			msg:  domain.WriterMessage{Type: "page_data", URL: "http://example.com", ScrapingID: 123, Links: []string{"l1", "l2"}},
			setupMocks: func(db *MockDBRepository, status *MockJobStatusRepository) {
				db.On("InsertPageData", mock.Anything, mock.MatchedBy(func(m domain.WriterMessage) bool { return m.URL == "http://example.com" })).Return(nil)
				status.On("IncrementLinkCount", mock.Anything, "123", 2).Return(nil)
			},
		},
		{
			name: "Image Explanation Success",
			msg:  domain.WriterMessage{Type: "image_explanation", URL: "http://img.com/1.jpg", Explanation: "A nice picture"},
			setupMocks: func(db *MockDBRepository, status *MockJobStatusRepository) {
				db.On("InsertImageExplanation", mock.Anything, mock.MatchedBy(func(m domain.WriterMessage) bool { return m.Explanation == "A nice picture" })).Return(nil)
			},
		},
		{
			name: "Page Summary Success",
			msg:  domain.WriterMessage{Type: "page_summary", URL: "http://example.com/page", Summary: "This is a summary", ScrapingID: 123},
			setupMocks: func(db *MockDBRepository, status *MockJobStatusRepository) {
				db.On("InsertPageSummary", mock.Anything, mock.MatchedBy(func(m domain.WriterMessage) bool { return m.Summary == "This is a summary" })).Return(nil)
			},
		},
		{
			name: "Scraping Complete Success",
			msg:  domain.WriterMessage{Type: "scraping_complete", ScrapingID: 123},
			setupMocks: func(db *MockDBRepository, status *MockJobStatusRepository) {
				db.On("CompleteScraping", mock.Anything, 123).Return(nil)
				status.On("UpdateJobStatusFull", mock.Anything, "123", domain.StatusCompleted, mock.Anything).Return(nil)
			},
		},
		{
			name: "Scraping Complete - Status Update Error (Logged, not returned)",
			msg:  domain.WriterMessage{Type: "scraping_complete", ScrapingID: 123},
			setupMocks: func(db *MockDBRepository, status *MockJobStatusRepository) {
				db.On("CompleteScraping", mock.Anything, 123).Return(nil)
				status.On("UpdateJobStatusFull", mock.Anything, "123", domain.StatusCompleted, mock.Anything).Return(assert.AnError)
			},
		},
		{
			name:          "Unknown Type - No Action",
			msg:           domain.WriterMessage{Type: "unknown"},
			expectNoCalls: true,
		},
		{
			name: "Repo Error - Propagated",
			msg:  domain.WriterMessage{Type: "page_data"},
			setupMocks: func(db *MockDBRepository, status *MockJobStatusRepository) {
				db.On("InsertPageData", mock.Anything, mock.Anything).Return(assert.AnError)
			},
			expectedError: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDb := new(MockDBRepository)
			mockStatus := new(MockJobStatusRepository)
			s := NewWriterService(
				WithDBRepository(mockDb),
				WithJobStatusRepository(mockStatus),
			)

			if tt.setupMocks != nil {
				tt.setupMocks(mockDb, mockStatus)
			}

			err := s.ProcessMessage(context.Background(), tt.msg)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectNoCalls {
				mockDb.AssertNotCalled(t, "InsertPageData", mock.Anything, mock.Anything)
				mockDb.AssertNotCalled(t, "InsertImageExplanation", mock.Anything, mock.Anything)
			} else {
				mockDb.AssertExpectations(t)
				mockStatus.AssertExpectations(t)
			}
		})
	}
}
