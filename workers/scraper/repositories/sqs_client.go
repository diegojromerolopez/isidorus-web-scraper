package repositories

import (
	"context"
	"encoding/json"
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

func (s *AWSSQSClient) ReceiveMessages(ctx context.Context, queueURL string, maxMessages int32) (*sqs.ReceiveMessageOutput, error) {
	out, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: maxMessages,
		WaitTimeSeconds:     20,
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

func (s *AWSSQSClient) SendMessage(ctx context.Context, queueURL string, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message for %s: %w", queueURL, err)
	}
	_, err = s.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("failed to send message to %s: %w", queueURL, err)
	}
	return nil
}
