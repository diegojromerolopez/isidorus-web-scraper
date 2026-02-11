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
	tests := []struct {
		name    string
		output  *sqs.ReceiveMessageOutput
		err     error
		wantErr bool
	}{
		{
			name: "Success",
			output: &sqs.ReceiveMessageOutput{
				Messages: []types.Message{{Body: aws.String(`{"url":"http://test.com"}`), ReceiptHandle: aws.String("h1")}},
			},
		},
		{
			name:    "Error",
			err:     errors.New("sqs error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
				o.APIOptions = append(o.APIOptions, mockSQSMiddleware(tt.output, tt.err))
			})
			repo := NewSQSRepository(client, "test-url")
			messages, handles, err := repo.ReceiveMessages(context.TODO())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.output.Messages), len(messages))
				assert.Equal(t, "h1", handles[0])
			}
		})
	}
}

func TestSQSRepository_DeleteMessage(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{
			name: "Success",
		},
		{
			name:    "Error",
			err:     errors.New("sqs error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
				o.APIOptions = append(o.APIOptions, mockSQSMiddleware(&sqs.DeleteMessageOutput{}, tt.err))
			})
			repo := NewSQSRepository(client, "test-url")
			err := repo.DeleteMessage(context.TODO(), "h1")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
