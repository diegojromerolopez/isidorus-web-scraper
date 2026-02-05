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

func TestSQSRepository_ReceiveMessages(t *testing.T) {
	output := &sqs.ReceiveMessageOutput{
		Messages: []types.Message{
			{Body: aws.String(`{"url":"http://test.com","content":"test","summary":"sum","scraping_id":1,"user_id":1}`), ReceiptHandle: aws.String("h1")},
		},
	}
	client := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
		o.APIOptions = append(o.APIOptions, mockSQSMiddleware(output, nil))
	})

	repo := NewSQSRepository(client, "test-url")
	messages, handles, err := repo.ReceiveMessages(context.TODO())

	assert.NoError(t, err)
	assert.Equal(t, 1, len(messages))
	assert.Equal(t, "h1", handles[0])
	assert.Equal(t, "http://test.com", messages[0].URL)
}

func TestSQSRepository_ReceiveMessages_Error(t *testing.T) {
	client := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
		o.APIOptions = append(o.APIOptions, mockSQSMiddleware(nil, errors.New("sqs error")))
	})

	repo := NewSQSRepository(client, "test-url")
	_, _, err := repo.ReceiveMessages(context.TODO())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to receive messages")
}

func TestSQSRepository_DeleteMessage(t *testing.T) {
	client := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
		o.APIOptions = append(o.APIOptions, mockSQSMiddleware(&sqs.DeleteMessageOutput{}, nil))
	})

	repo := NewSQSRepository(client, "test-url")
	err := repo.DeleteMessage(context.TODO(), "h1")

	assert.NoError(t, err)
}
