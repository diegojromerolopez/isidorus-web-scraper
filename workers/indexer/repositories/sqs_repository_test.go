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
	"workers/indexer/domain"
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
		name     string
		output   *sqs.ReceiveMessageOutput
		err      error
		wantErr  bool
		assertFn func(*testing.T, []domain.IndexMessage)
	}{
		{
			name: "Success",
			output: &sqs.ReceiveMessageOutput{
				Messages: []types.Message{{Body: aws.String(`{"url":"http://test.com"}`), ReceiptHandle: aws.String("h1")}},
			},
		},
		{
			name: "SuccessWithTraceAttributes",
			output: &sqs.ReceiveMessageOutput{
				Messages: []types.Message{
					{
						Body:          aws.String(`{"url":"http://test.com"}`),
						ReceiptHandle: aws.String("h2"),
						MessageAttributes: map[string]types.MessageAttributeValue{
							"traceparent": {
								DataType:    aws.String("String"),
								StringValue: aws.String("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"),
							},
						},
					},
				},
			},
			assertFn: func(t *testing.T, msgs []domain.IndexMessage) {
				assert.NotEmpty(t, msgs)
				assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", msgs[0].TraceContext["traceparent"])
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
			repo := NewSQSRepository(client, "test-url", NewNoopTelemetryClient())
			messages, handles, err := repo.ReceiveMessages(context.TODO())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.output.Messages), len(messages))
				if tt.name == "Success" {
					assert.Equal(t, "h1", handles[0])
				} else if tt.name == "SuccessWithTraceAttributes" {
					assert.Equal(t, "h2", handles[0])
				}
				if tt.assertFn != nil {
					tt.assertFn(t, messages)
				}
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
			repo := NewSQSRepository(client, "test-url", NewNoopTelemetryClient())
			err := repo.DeleteMessage(context.TODO(), "h1")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
