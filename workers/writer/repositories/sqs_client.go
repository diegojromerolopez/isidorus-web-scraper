package repositories

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type AWSSQSClient struct {
	client *sqs.Client
}

func NewSQSClient(client *sqs.Client) *AWSSQSClient {
	return &AWSSQSClient{client: client}
}

func (s *AWSSQSClient) ReceiveMessages(ctx context.Context, queueURL string, maxMessages int32, waitTime int32) (*sqs.ReceiveMessageOutput, error) {
	out, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: maxMessages,
		WaitTimeSeconds:     waitTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to receive messages from %s: %w", queueURL, err)
	}
	return out, nil
}

func (s *AWSSQSClient) DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error {
	_, err := s.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: receiptHandle,
	})
	if err != nil {
		return fmt.Errorf("failed to delete message from %s: %w", queueURL, err)
	}
	return nil
}

func (s *AWSSQSClient) DeleteMessageBatch(ctx context.Context, queueURL string, entries []types.DeleteMessageBatchRequestEntry) error {
	_, err := s.client.DeleteMessageBatch(ctx, &sqs.DeleteMessageBatchInput{
		QueueUrl: aws.String(queueURL),
		Entries:  entries,
	})
	if err != nil {
		return fmt.Errorf("failed to batch delete messages from %s: %w", queueURL, err)
	}
	return nil
}
