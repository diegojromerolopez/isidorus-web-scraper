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
	res, err := repo.ReceiveMessages(context.TODO(), "queue-url", 10, 20)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(res.Messages))

	// Error case
	clientErr := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
		o.APIOptions = append(o.APIOptions, mockSQSMiddleware(nil, errors.New("aws error")))
	})

	repoErr := NewSQSClient(clientErr)
	_, err = repoErr.ReceiveMessages(context.TODO(), "queue-url", 10, 20)
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
