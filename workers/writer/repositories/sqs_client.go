package repositories

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type AWSSQSClient struct {
	client     *sqs.Client
	otelClient TelemetryClient
}

func NewSQSClient(client *sqs.Client, otelClient TelemetryClient) *AWSSQSClient {
	return &AWSSQSClient{client: client, otelClient: otelClient}
}

func (s *AWSSQSClient) ReceiveMessages(ctx context.Context, queueURL string, maxMessages int32, waitTime int32) (*sqs.ReceiveMessageOutput, error) {
	ctx, span := s.otelClient.StartSpan(ctx, "AWSSQSClient.ReceiveMessages",
		WithAttribute("queueURL", queueURL),
		WithAttribute("maxMessages", maxMessages),
		WithAttribute("waitTime", waitTime),
	)
	defer span.End()

	out, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(queueURL),
		MaxNumberOfMessages:   maxMessages,
		WaitTimeSeconds:       waitTime,
		MessageAttributeNames: []string{"All"},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return nil, fmt.Errorf("failed to receive messages from %s: %w", queueURL, err)
	}
	span.SetStatus("ok", "success")
	return out, nil
}

func (s *AWSSQSClient) DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error {
	ctx, span := s.otelClient.StartSpan(ctx, "AWSSQSClient.DeleteMessage",
		WithAttribute("queueURL", queueURL),
	)
	defer span.End()

	_, err := s.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: receiptHandle,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("failed to delete message from %s: %w", queueURL, err)
	}
	span.SetStatus("ok", "success")
	return nil
}
