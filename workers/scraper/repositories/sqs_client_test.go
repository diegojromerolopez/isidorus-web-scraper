package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"
)

// Mock middleware to return specific output or error
func mockSQSMiddleware(output interface{}, err error) func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		return stack.Finalize.Add(
			middleware.FinalizeMiddlewareFunc("MockMiddleware", func(context.Context, middleware.FinalizeInput, middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
				return middleware.FinalizeOutput{
					Result: output,
				}, middleware.Metadata{}, err
			}),
			middleware.Before,
		)
	}
}

func TestSQSClient_SendMessage(t *testing.T) {
	// Success case
	client := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
		o.APIOptions = append(o.APIOptions, mockSQSMiddleware(&sqs.SendMessageOutput{}, nil))
	})

	repo := NewSQSClient(client)
	err := repo.SendMessage(context.TODO(), "queue-url", map[string]string{"key": "value"})
	assert.NoError(t, err)

	// Error case
	clientErr := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
		o.APIOptions = append(o.APIOptions, mockSQSMiddleware(nil, errors.New("aws error")))
	})

	repoErr := NewSQSClient(clientErr)
	err = repoErr.SendMessage(context.TODO(), "queue-url", map[string]string{"key": "value"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send message")
}

func TestSQSClient_ReceiveMessages(t *testing.T) {
	// Success case
	output := &sqs.ReceiveMessageOutput{
		Messages: []types.Message{
			{Body: aws.String(`{"key":"value"}`), ReceiptHandle: aws.String("handle")},
		},
	}
	client := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
		o.APIOptions = append(o.APIOptions, mockSQSMiddleware(output, nil))
	})

	repo := NewSQSClient(client)
	res, err := repo.ReceiveMessages(context.TODO(), "queue-url")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(res.Messages))

	// Error case
	clientErr := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
		o.APIOptions = append(o.APIOptions, mockSQSMiddleware(nil, errors.New("aws error")))
	})

	repoErr := NewSQSClient(clientErr)
	_, err = repoErr.ReceiveMessages(context.TODO(), "queue-url")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to receive messages")
}

func TestSQSClient_DeleteMessage(t *testing.T) {
	// Success case
	client := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
		o.APIOptions = append(o.APIOptions, mockSQSMiddleware(&sqs.DeleteMessageOutput{}, nil))
	})

	repo := NewSQSClient(client)
	handle := "receipt-handle"
	err := repo.DeleteMessage(context.TODO(), "queue-url", &handle)
	assert.NoError(t, err)

	// Error case
	clientErr := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
		o.APIOptions = append(o.APIOptions, mockSQSMiddleware(nil, errors.New("aws error")))
	})

	repoErr := NewSQSClient(clientErr)
	err = repoErr.DeleteMessage(context.TODO(), "queue-url", &handle)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete message")
}

func TestSQSClient_SendMessage_MarshalError(t *testing.T) {
	client := sqs.NewFromConfig(aws.Config{})
	repo := NewSQSClient(client)

	// Channel cannot be marshaled to JSON
	msg := map[string]interface{}{
		"key": make(chan int),
	}

	err := repo.SendMessage(context.TODO(), "queue-url", msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal message")
}
