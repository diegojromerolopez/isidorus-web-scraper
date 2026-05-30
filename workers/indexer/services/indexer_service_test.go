package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"workers/indexer/domain"
)

type MockSQSRepository struct {
	ReceiveMessagesFunc func(ctx context.Context) ([]domain.IndexMessage, []string, error)
	DeleteMessageFunc   func(ctx context.Context, handle string) error
	ReceiveCalled       int
	DeleteCalled        int
}

func (m *MockSQSRepository) ReceiveMessages(ctx context.Context) ([]domain.IndexMessage, []string, error) {
	m.ReceiveCalled++
	if m.ReceiveMessagesFunc != nil {
		return m.ReceiveMessagesFunc(ctx)
	}
	return nil, nil, nil
}

func (m *MockSQSRepository) DeleteMessage(ctx context.Context, handle string) error {
	m.DeleteCalled++
	if m.DeleteMessageFunc != nil {
		return m.DeleteMessageFunc(ctx, handle)
	}
	return nil
}

type MockOpenSearchRepository struct {
	IndexDocumentFunc func(ctx context.Context, msg domain.IndexMessage) error
	IndexCalled       int
}

func (m *MockOpenSearchRepository) IndexDocument(ctx context.Context, msg domain.IndexMessage) error {
	m.IndexCalled++
	if m.IndexDocumentFunc != nil {
		return m.IndexDocumentFunc(ctx, msg)
	}
	return nil
}

func TestIndexerService_Start(t *testing.T) {
	tests := []struct {
		name          string
		sqsRepo       *MockSQSRepository
		osRepo        *MockOpenSearchRepository
		expectedIndex int
		expectedDel   int
	}{
		{
			name: "Success Path",
			sqsRepo: &MockSQSRepository{
				ReceiveMessagesFunc: func(ctx context.Context) ([]domain.IndexMessage, []string, error) {
					return []domain.IndexMessage{{URL: "http://site.com", Content: "text"}}, []string{"h1"}, nil
				},
			},
			osRepo: &MockOpenSearchRepository{
				IndexDocumentFunc: func(ctx context.Context, msg domain.IndexMessage) error { return nil },
			},
			expectedIndex: 1,
			expectedDel:   1,
		},
		{
			name: "Index Error - No Delete",
			sqsRepo: &MockSQSRepository{
				ReceiveMessagesFunc: func(ctx context.Context) ([]domain.IndexMessage, []string, error) {
					return []domain.IndexMessage{{URL: "http://err.com"}}, []string{"h1"}, nil
				},
			},
			osRepo: &MockOpenSearchRepository{
				IndexDocumentFunc: func(ctx context.Context, msg domain.IndexMessage) error { return errors.New("os error") },
			},
			expectedIndex: 1,
			expectedDel:   0,
		},
		{
			name: "Receive Error",
			sqsRepo: &MockSQSRepository{
				ReceiveMessagesFunc: func(ctx context.Context) ([]domain.IndexMessage, []string, error) {
					return nil, nil, errors.New("sqs error")
				},
			},
			osRepo:        &MockOpenSearchRepository{},
			expectedIndex: 0,
			expectedDel:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewIndexerService(tt.sqsRepo, tt.osRepo, nil)
			service.retryDelay = 1 * time.Millisecond

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			go service.Start(ctx)

			time.Sleep(100 * time.Millisecond)

			assert.GreaterOrEqual(t, tt.osRepo.IndexCalled, tt.expectedIndex)
			assert.GreaterOrEqual(t, tt.sqsRepo.DeleteCalled, tt.expectedDel)
		})
	}
}
