package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"indexer-worker/domain"
)

type MockSQSRepository struct {
	ReceiveMessagesFunc func(ctx context.Context) ([]domain.IndexMessage, []string, error)
	DeleteMessageFunc   func(ctx context.Context, handle string) error
	ReceiveCalled       int
	DeleteCalled        int
}

func (m *MockSQSRepository) ReceiveMessages(ctx context.Context) ([]domain.IndexMessage, []string, error) {
	m.ReceiveCalled++
	return m.ReceiveMessagesFunc(ctx)
}

func (m *MockSQSRepository) DeleteMessage(ctx context.Context, handle string) error {
	m.DeleteCalled++
	return m.DeleteMessageFunc(ctx, handle)
}

type MockOpenSearchRepository struct {
	IndexDocumentFunc func(ctx context.Context, msg domain.IndexMessage) error
	IndexCalled       int
}

func (m *MockOpenSearchRepository) IndexDocument(ctx context.Context, msg domain.IndexMessage) error {
	m.IndexCalled++
	return m.IndexDocumentFunc(ctx, msg)
}

func TestIndexerService_Start_ProcessMessageSuccessfully(t *testing.T) {
	sqsRepo := &MockSQSRepository{
		ReceiveMessagesFunc: func(ctx context.Context) ([]domain.IndexMessage, []string, error) {
			if ctx.Err() != nil {
				return nil, nil, ctx.Err()
			}
			return []domain.IndexMessage{
				{URL: "http://example.com", Content: "text", Summary: "sum", ScrapingID: 1, UserID: 1},
			}, []string{"handle1"}, nil
		},
		DeleteMessageFunc: func(ctx context.Context, handle string) error {
			return nil
		},
	}

	osRepo := &MockOpenSearchRepository{
		IndexDocumentFunc: func(ctx context.Context, msg domain.IndexMessage) error {
			return nil
		},
	}

	service := NewIndexerService(sqsRepo, osRepo)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run in a goroutine because Start is an infinite loop
	go service.Start(ctx)

	// Wait for a bit to let it process
	time.Sleep(200 * time.Millisecond)

	assert.GreaterOrEqual(t, sqsRepo.ReceiveCalled, 1)
	assert.GreaterOrEqual(t, osRepo.IndexCalled, 1)
	assert.GreaterOrEqual(t, sqsRepo.DeleteCalled, 1)
}

func TestIndexerService_Start_IndexError(t *testing.T) {
	sqsRepo := &MockSQSRepository{
		ReceiveMessagesFunc: func(ctx context.Context) ([]domain.IndexMessage, []string, error) {
			return []domain.IndexMessage{
				{URL: "http://example.com", Content: "text", Summary: "sum", ScrapingID: 1, UserID: 1},
			}, []string{"handle1"}, nil
		},
		DeleteMessageFunc: func(ctx context.Context, handle string) error {
			return nil
		},
	}

	osRepo := &MockOpenSearchRepository{
		IndexDocumentFunc: func(ctx context.Context, msg domain.IndexMessage) error {
			return errors.New("os error")
		},
	}

	service := NewIndexerService(sqsRepo, osRepo)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go service.Start(ctx)

	time.Sleep(200 * time.Millisecond)

	assert.GreaterOrEqual(t, osRepo.IndexCalled, 1)
	assert.Equal(t, 0, sqsRepo.DeleteCalled) // Should not delete if index failed
}
