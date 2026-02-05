package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"indexer-worker/domain"
)

type SQSRepository struct {
	client   *sqs.Client
	queueURL string
}

func NewSQSRepository(client *sqs.Client, queueURL string) *SQSRepository {
	return &SQSRepository{
		client:   client,
		queueURL: queueURL,
	}
}

func (r *SQSRepository) ReceiveMessages(ctx context.Context) ([]domain.IndexMessage, []string, error) {
	output, err := r.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(r.queueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20,
	})
	if err != nil {
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
		messages = append(messages, indexMsg)
		handles = append(handles, *msg.ReceiptHandle)
	}

	return messages, handles, nil
}

func (r *SQSRepository) DeleteMessage(ctx context.Context, handle string) error {
	_, err := r.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(r.queueURL),
		ReceiptHandle: aws.String(handle),
	})
	return err
}
