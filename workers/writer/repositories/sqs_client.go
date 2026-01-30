package repositories

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSClient interface {
	ReceiveMessages(ctx context.Context, queueURL string, maxMessages int32, waitTime int32) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error
}

type AWSSQSClient struct {
	Client *sqs.Client
}

func NewSQSClient(client *sqs.Client) SQSClient {
	return &AWSSQSClient{Client: client}
}

func (s *AWSSQSClient) ReceiveMessages(ctx context.Context, queueURL string, maxMessages int32, waitTime int32) (*sqs.ReceiveMessageOutput, error) {
	return s.Client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: maxMessages,
		WaitTimeSeconds:     waitTime,
	})
}

func (s *AWSSQSClient) DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error {
	_, err := s.Client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: receiptHandle,
	})
	return err
}
