package repositories

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSRepository struct {
	client     *sqs.Client
	otelClient TelemetryClient
}

func NewSQSRepository(cfg aws.Config, otelClient TelemetryClient) *SQSRepository {
	return &SQSRepository{
		client:     sqs.NewFromConfig(cfg),
		otelClient: otelClient,
	}
}

func (r *SQSRepository) ReceiveMessages(ctx context.Context, queueURL string) ([]types.Message, error) {
	ctx, span := r.otelClient.StartSpan(ctx, "SQSRepository.ReceiveMessages",
		WithAttribute("queueURL", queueURL),
	)
	defer span.End()

	output, err := r.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return nil, err
	}
	span.SetStatus("ok", "success")
	return output.Messages, nil
}

func (r *SQSRepository) SendMessage(ctx context.Context, queueURL string, body interface{}) error {
	ctx, span := r.otelClient.StartSpan(ctx, "SQSRepository.SendMessage",
		WithAttribute("queueURL", queueURL),
	)
	defer span.End()

	jsonBody, err := json.Marshal(body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return err
	}
	_, err = r.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(jsonBody)),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return err
	}
	span.SetStatus("ok", "success")
	return nil
}

func (r *SQSRepository) DeleteMessage(ctx context.Context, queueURL string, receiptHandle string) error {
	ctx, span := r.otelClient.StartSpan(ctx, "SQSRepository.DeleteMessage",
		WithAttribute("queueURL", queueURL),
	)
	defer span.End()

	_, err := r.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return err
	}
	span.SetStatus("ok", "success")
	return nil
}
