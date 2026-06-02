package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type AWSSQSClient struct {
	client     *sqs.Client
	otelClient TelemetryClient
}

func NewSQSClient(client *sqs.Client, otelClient TelemetryClient) *AWSSQSClient {
	return &AWSSQSClient{client: client, otelClient: otelClient}
}

func (s *AWSSQSClient) ReceiveMessages(ctx context.Context, queueURL string) (*sqs.ReceiveMessageOutput, error) {
	ctx, span := s.otelClient.StartSpan(ctx, "AWSSQSClient.ReceiveMessages", WithAttribute("queueURL", queueURL))
	defer span.End()

	out, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(queueURL),
		MaxNumberOfMessages:   1,
		WaitTimeSeconds:       20,
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
	ctx, span := s.otelClient.StartSpan(ctx, "AWSSQSClient.DeleteMessage", WithAttribute("queueURL", queueURL))
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

func (s *AWSSQSClient) SendMessage(ctx context.Context, queueURL string, msg interface{}) error {
	ctx, span := s.otelClient.StartSpan(ctx, "AWSSQSClient.SendMessage", WithAttribute("queueURL", queueURL))
	defer span.End()

	body, err := json.Marshal(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("failed to marshal message for %s: %w", queueURL, err)
	}

	// Inject standard trace context as SQS Message Attributes
	traceMap := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(traceMap))

	msgAttrs := make(map[string]types.MessageAttributeValue)
	for k, v := range traceMap {
		msgAttrs[k] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(v),
		}
	}

	_, err = s.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:          aws.String(queueURL),
		MessageBody:       aws.String(string(body)),
		MessageAttributes: msgAttrs,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("failed to send message to %s: %w", queueURL, err)
	}
	span.SetStatus("ok", "success")
	return nil
}
