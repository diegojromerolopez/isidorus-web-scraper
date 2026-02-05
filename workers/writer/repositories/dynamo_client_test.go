package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDynamoDB struct {
	mock.Mock
}

func (m *MockDynamoDB) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.UpdateItemOutput), args.Error(1)
}

func TestNewDynamoDBClient(t *testing.T) {
	client := NewDynamoDBClient(nil, "test-table")
	assert.NotNil(t, client)
	assert.Equal(t, "test-table", client.tableName)
}

func TestUpdateJobStatus_NoTable(t *testing.T) {
	client := NewDynamoDBClient(nil, "")
	err := client.UpdateJobStatus(context.Background(), "123", "PENDING")
	assert.NoError(t, err)
}

func TestUpdateJobStatus_Success(t *testing.T) {
	mockDB := new(MockDynamoDB)
	client := NewDynamoDBClient(mockDB, "test-table")

	mockDB.On("UpdateItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.UpdateItemInput) bool {
		return *input.TableName == "test-table" && input.Key["scraping_id"] != nil
	}), mock.Anything).Return(&dynamodb.UpdateItemOutput{}, nil)

	err := client.UpdateJobStatus(context.Background(), "123", "PENDING")
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestUpdateJobStatus_Error(t *testing.T) {
	mockDB := new(MockDynamoDB)
	client := NewDynamoDBClient(mockDB, "test-table")

	mockDB.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("dynamo error"))

	err := client.UpdateJobStatus(context.Background(), "123", "PENDING")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update job status")
}

func TestUpdateJobStatusFull_NoTable(t *testing.T) {
	client := NewDynamoDBClient(nil, "")
	err := client.UpdateJobStatusFull(context.Background(), "123", "COMPLETED", "2024-01-01")
	assert.NoError(t, err)
}

func TestUpdateJobStatusFull_Success(t *testing.T) {
	mockDB := new(MockDynamoDB)
	client := NewDynamoDBClient(mockDB, "test-table")

	mockDB.On("UpdateItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.UpdateItemInput) bool {
		return *input.TableName == "test-table" && input.Key["scraping_id"] != nil
	}), mock.Anything).Return(&dynamodb.UpdateItemOutput{}, nil)

	err := client.UpdateJobStatusFull(context.Background(), "123", "COMPLETED", "2024-01-01")
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestUpdateJobStatusFull_Error(t *testing.T) {
	mockDB := new(MockDynamoDB)
	client := NewDynamoDBClient(mockDB, "test-table")

	mockDB.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("dynamo error"))

	err := client.UpdateJobStatusFull(context.Background(), "123", "COMPLETED", "2024-01-01")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update job status full")
}
