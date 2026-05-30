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
	tests := []struct {
		name    string
		output  interface{}
		err     error
		body    interface{}
		wantErr bool
	}{
		{
			name:   "Success",
			output: &sqs.SendMessageOutput{},
			body:   map[string]string{"key": "value"},
		},
		{
			name:    "AWS Error",
			err:     errors.New("aws error"),
			body:    map[string]string{"key": "value"},
			wantErr: true,
		},
		{
			name:    "Marshal Error",
			body:    map[string]interface{}{"key": make(chan int)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
				if tt.body != nil && tt.name != "Marshal Error" {
					o.APIOptions = append(o.APIOptions, mockSQSMiddleware(tt.output, tt.err))
				}
			})
			repo := NewSQSClient(client, NewNoopTelemetryClient())
			err := repo.SendMessage(context.TODO(), "queue-url", tt.body)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSQSClient_ReceiveMessages(t *testing.T) {
	tests := []struct {
		name    string
		output  *sqs.ReceiveMessageOutput
		err     error
		wantErr bool
	}{
		{
			name: "Success",
			output: &sqs.ReceiveMessageOutput{
				Messages: []types.Message{{Body: aws.String(`{"key":"value"}`), ReceiptHandle: aws.String("handle")}},
			},
		},
		{
			name:    "AWS Error",
			err:     errors.New("aws error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
				o.APIOptions = append(o.APIOptions, mockSQSMiddleware(tt.output, tt.err))
			})
			repo := NewSQSClient(client, NewNoopTelemetryClient())
			res, err := repo.ReceiveMessages(context.TODO(), "queue-url")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.output.Messages), len(res.Messages))
			}
		})
	}
}

func TestSQSClient_DeleteMessage(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{
			name: "Success",
		},
		{
			name:    "AWS Error",
			err:     errors.New("aws error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := sqs.NewFromConfig(aws.Config{}, func(o *sqs.Options) {
				o.APIOptions = append(o.APIOptions, mockSQSMiddleware(&sqs.DeleteMessageOutput{}, tt.err))
			})
			repo := NewSQSClient(client, NewNoopTelemetryClient())
			handle := "receipt-handle"
			err := repo.DeleteMessage(context.TODO(), "queue-url", &handle)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
