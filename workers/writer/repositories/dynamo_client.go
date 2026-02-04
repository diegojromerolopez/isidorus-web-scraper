package repositories

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBAPI interface {
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

type DynamoDBClient struct {
	client    DynamoDBAPI
	tableName string
}

func NewDynamoDBClient(client DynamoDBAPI, tableName string) *DynamoDBClient {
	return &DynamoDBClient{
		client:    client,
		tableName: tableName,
	}
}

func (d *DynamoDBClient) UpdateJobStatus(ctx context.Context, jobID string, status string) error {
	if d.tableName == "" {
		log.Printf("Warning: DYNAMODB_TABLE not configured, skipping DynamoDB status update for job %s", jobID)
		return nil
	}

	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"scraping_id": &types.AttributeValueMemberS{Value: jobID},
		},
		UpdateExpression: aws.String("SET #s = :status"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: status},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to update job status in DynamoDB for job %s: %w", jobID, err)
	}

	log.Printf("Successfully updated job %s status to %s in DynamoDB (Table: %s)", jobID, status, d.tableName)
	return nil
}
func (d *DynamoDBClient) UpdateJobStatusFull(ctx context.Context, jobID string, status string, completedAt string) error {
	if d.tableName == "" {
		log.Printf("Warning: DYNAMODB_TABLE not configured, skipping DynamoDB status update for job %s", jobID)
		return nil
	}

	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"scraping_id": &types.AttributeValueMemberS{Value: jobID},
		},
		UpdateExpression: aws.String("SET #s = :status, completed_at = :cat"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: status},
			":cat":    &types.AttributeValueMemberS{Value: completedAt},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to update job status full in DynamoDB for job %s: %w", jobID, err)
	}

	log.Printf("Successfully updated job %s status to %s and completed_at to %s in DynamoDB (Table: %s)", jobID, status, completedAt, d.tableName)
	return nil
}

func (d *DynamoDBClient) IncrementLinkCount(ctx context.Context, jobID string, increment int) error {
	if d.tableName == "" {
		return nil
	}

	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"scraping_id": &types.AttributeValueMemberS{Value: jobID},
		},
		UpdateExpression: aws.String("ADD links_count :inc"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inc": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", increment)},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to increment link count in DynamoDB for job %s: %w", jobID, err)
	}

	return nil
}
