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
	client     DynamoDBAPI
	tableName  string
	otelClient TelemetryClient
}

func NewDynamoDBClient(client DynamoDBAPI, tableName string, otelClient TelemetryClient) *DynamoDBClient {
	return &DynamoDBClient{
		client:     client,
		tableName:  tableName,
		otelClient: otelClient,
	}
}

func (d *DynamoDBClient) UpdateJobStatus(ctx context.Context, jobID string, status string) error {
	ctx, span := d.otelClient.StartSpan(ctx, "DynamoDBClient.UpdateJobStatus",
		WithAttribute("jobID", jobID),
		WithAttribute("status", status),
	)
	defer span.End()

	if d.tableName == "" {
		log.Printf("Warning: DYNAMODB_TABLE not configured, skipping DynamoDB status update for job %s", jobID)
		span.SetStatus("ok", "skipped (no table)")
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
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("failed to update job status in DynamoDB for job %s: %w", jobID, err)
	}

	log.Printf("Successfully updated job %s status to %s in DynamoDB (Table: %s)", jobID, status, d.tableName)
	span.SetStatus("ok", "success")
	return nil
}

func (d *DynamoDBClient) UpdateJobStatusFull(ctx context.Context, jobID string, status string, completedAt string) error {
	ctx, span := d.otelClient.StartSpan(ctx, "DynamoDBClient.UpdateJobStatusFull",
		WithAttribute("jobID", jobID),
		WithAttribute("status", status),
		WithAttribute("completedAt", completedAt),
	)
	defer span.End()

	if d.tableName == "" {
		log.Printf("Warning: DYNAMODB_TABLE not configured, skipping DynamoDB status update for job %s", jobID)
		span.SetStatus("ok", "skipped (no table)")
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
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("failed to update job status full in DynamoDB for job %s: %w", jobID, err)
	}

	log.Printf("Successfully updated job %s status to %s and completed_at to %s in DynamoDB (Table: %s)", jobID, status, completedAt, d.tableName)
	span.SetStatus("ok", "success")
	return nil
}

func (d *DynamoDBClient) IncrementLinkCount(ctx context.Context, jobID string, increment int) error {
	ctx, span := d.otelClient.StartSpan(ctx, "DynamoDBClient.IncrementLinkCount",
		WithAttribute("jobID", jobID),
		WithAttribute("increment", increment),
	)
	defer span.End()

	if d.tableName == "" {
		span.SetStatus("ok", "skipped (no table)")
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
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return fmt.Errorf("failed to increment link count in DynamoDB for job %s: %w", jobID, err)
	}

	span.SetStatus("ok", "success")
	return nil
}
