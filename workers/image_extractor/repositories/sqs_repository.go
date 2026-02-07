package repositories

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSRepository struct {
	client *sqs.Client
}

func NewSQSRepository(cfg aws.Config) *SQSRepository {
	return &SQSRepository{
		client: sqs.NewFromConfig(cfg),
	}
}

func (r *SQSRepository) ReceiveMessages(ctx context.Context, queueURL string) ([]types.Message, error) {
	output, err := r.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20,
	})
	if err != nil {
		return nil, err
	}
	return output.Messages, nil
}

func (r *SQSRepository) SendMessage(ctx context.Context, queueURL string, body interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = r.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(jsonBody)),
	})
	return err
}

func (r *SQSRepository) DeleteMessage(ctx context.Context, queueURL string, receiptHandle string) error {
	_, err := r.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	return err
}
