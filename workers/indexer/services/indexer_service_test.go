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
	// Reduce delay for quicker tests if needed, though not used in success path
	service.retryDelay = 1 * time.Millisecond

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
	service.retryDelay = 1 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go service.Start(ctx)

	time.Sleep(200 * time.Millisecond)

	assert.GreaterOrEqual(t, osRepo.IndexCalled, 1)
	assert.Equal(t, 0, sqsRepo.DeleteCalled) // Should not delete if index failed
}

func TestIndexerService_Start_ReceiveError(t *testing.T) {
	sqsRepo := &MockSQSRepository{
		ReceiveMessagesFunc: func(ctx context.Context) ([]domain.IndexMessage, []string, error) {
			// Return error to trigger the error handling path
			return nil, nil, errors.New("sqs receive error")
		},
	}

	osRepo := &MockOpenSearchRepository{}

	service := NewIndexerService(sqsRepo, osRepo)
	service.retryDelay = 1 * time.Millisecond // fast retry

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	go service.Start(ctx)

	time.Sleep(100 * time.Millisecond)

	assert.GreaterOrEqual(t, sqsRepo.ReceiveCalled, 1)
	assert.Equal(t, 0, osRepo.IndexCalled)
}

func TestIndexerService_Start_DeleteError(t *testing.T) {
	sqsRepo := &MockSQSRepository{
		ReceiveMessagesFunc: func(ctx context.Context) ([]domain.IndexMessage, []string, error) {
			return []domain.IndexMessage{
				{URL: "http://example.com", Content: "text"},
			}, []string{"handle1"}, nil
		},
		DeleteMessageFunc: func(ctx context.Context, handle string) error {
			return errors.New("delete error")
		},
	}

	osRepo := &MockOpenSearchRepository{
		IndexDocumentFunc: func(ctx context.Context, msg domain.IndexMessage) error {
			return nil
		},
	}

	service := NewIndexerService(sqsRepo, osRepo)
	service.retryDelay = 1 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go service.Start(ctx)

	time.Sleep(200 * time.Millisecond)

	assert.GreaterOrEqual(t, sqsRepo.ReceiveCalled, 1)
	assert.GreaterOrEqual(t, osRepo.IndexCalled, 1)
	assert.GreaterOrEqual(t, sqsRepo.DeleteCalled, 1)
}
