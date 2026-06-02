package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"workers/indexer/domain"
)

type SQSRepository struct {
	client     *sqs.Client
	queueURL   string
	otelClient TelemetryClient
}

func NewSQSRepository(client *sqs.Client, queueURL string, otelClient TelemetryClient) *SQSRepository {
	return &SQSRepository{
		client:     client,
		queueURL:   queueURL,
		otelClient: otelClient,
	}
}

func (r *SQSRepository) ReceiveMessages(ctx context.Context) ([]domain.IndexMessage, []string, error) {
	ctx, span := r.otelClient.StartSpan(ctx, "SQSRepository.ReceiveMessages",
		WithAttribute("queueURL", r.queueURL),
	)
	defer span.End()

	output, err := r.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(r.queueURL),
		MaxNumberOfMessages:   10,
		WaitTimeSeconds:       20,
		MessageAttributeNames: []string{"All"},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return nil, nil, fmt.Errorf("failed to receive messages: %w", err)
	}

	var messages []domain.IndexMessage
	var handles []string
	for _, msg := range output.Messages {
		var indexMsg domain.IndexMessage
		if err := json.Unmarshal([]byte(*msg.Body), &indexMsg); err != nil {
			// Skip invalid messages but log them
			fmt.Printf("Received invalid message: %v\n", err)
			continue
		}
		// Extract trace context from SQS MessageAttributes
		traceContext := make(map[string]string)
		for k, attr := range msg.MessageAttributes {
			if attr.StringValue != nil {
				traceContext[k] = *attr.StringValue
			}
		}
		if len(traceContext) > 0 {
			indexMsg.TraceContext = traceContext
		}
		messages = append(messages, indexMsg)
		handles = append(handles, *msg.ReceiptHandle)
	}

	span.SetStatus("ok", "success")
	return messages, handles, nil
}

func (r *SQSRepository) DeleteMessage(ctx context.Context, handle string) error {
	ctx, span := r.otelClient.StartSpan(ctx, "SQSRepository.DeleteMessage",
		WithAttribute("queueURL", r.queueURL),
	)
	defer span.End()

	_, err := r.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(r.queueURL),
		ReceiptHandle: aws.String(handle),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return err
	}
	span.SetStatus("ok", "success")
	return nil
}
