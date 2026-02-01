package repositories

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSClient interface {
	ReceiveMessages(ctx context.Context, queueURL string) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error
	SendMessage(ctx context.Context, queueURL string, msg interface{}) error
}

type AWSSQSClient struct {
	Client *sqs.Client
}

func NewSQSClient(client *sqs.Client) SQSClient {
	return &AWSSQSClient{Client: client}
}

func (s *AWSSQSClient) ReceiveMessages(ctx context.Context, queueURL string) (*sqs.ReceiveMessageOutput, error) {
	return s.Client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     20,
	})
}

func (s *AWSSQSClient) DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error {
	_, err := s.Client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: receiptHandle,
	})
	return err
}

func (s *AWSSQSClient) SendMessage(ctx context.Context, queueURL string, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = s.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		log.Printf("failed to send message to %s: %v", queueURL, err)
		return err
	}
	return nil
}
