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
	client := NewDynamoDBClient(nil, "test-table", NewNoopTelemetryClient())
	assert.NotNil(t, client)
	assert.Equal(t, "test-table", client.tableName)
}

func TestDynamoDBClient_UpdateJobStatus(t *testing.T) {
	tests := []struct {
		name       string
		tableName  string
		scrapingID string
		status     string
		mockFunc   func(*MockDynamoDB)
		wantErr    bool
	}{
		{
			name:       "Success",
			tableName:  "test-table",
			scrapingID: "123",
			status:     "PENDING",
			mockFunc: func(m *MockDynamoDB) {
				m.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).Return(&dynamodb.UpdateItemOutput{}, nil)
			},
		},
		{
			name:       "No Table - NoOp",
			tableName:  "",
			scrapingID: "123",
			status:     "PENDING",
			mockFunc:   func(m *MockDynamoDB) {},
		},
		{
			name:       "Error",
			tableName:  "test-table",
			scrapingID: "123",
			status:     "PENDING",
			mockFunc: func(m *MockDynamoDB) {
				m.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("dynamo error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDynamoDB)
			tt.mockFunc(mockDB)
			client := NewDynamoDBClient(mockDB, tt.tableName, NewNoopTelemetryClient())
			if tt.tableName == "" {
				client.client = nil
			}

			err := client.UpdateJobStatus(context.Background(), tt.scrapingID, tt.status)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockDB.AssertExpectations(t)
		})
	}
}

func TestDynamoDBClient_UpdateJobStatusFull(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		mockFunc  func(*MockDynamoDB)
		wantErr   bool
	}{
		{
			name:      "Success",
			tableName: "test-table",
			mockFunc: func(m *MockDynamoDB) {
				m.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).Return(&dynamodb.UpdateItemOutput{}, nil)
			},
		},
		{
			name:      "Error",
			tableName: "test-table",
			mockFunc: func(m *MockDynamoDB) {
				m.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("dynamo error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDynamoDB)
			tt.mockFunc(mockDB)
			client := NewDynamoDBClient(mockDB, tt.tableName, NewNoopTelemetryClient())

			err := client.UpdateJobStatusFull(context.Background(), "123", "COMPLETED", "2024-01-01")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockDB.AssertExpectations(t)
		})
	}
}

func TestDynamoDBClient_IncrementLinkCount(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		mockFunc  func(*MockDynamoDB)
		wantErr   bool
	}{
		{
			name:      "Success",
			tableName: "test-table",
			mockFunc: func(m *MockDynamoDB) {
				m.On("UpdateItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.UpdateItemInput) bool {
					return *input.UpdateExpression == "ADD links_count :inc"
				}), mock.Anything).Return(&dynamodb.UpdateItemOutput{}, nil)
			},
		},
		{
			name:      "Error",
			tableName: "test-table",
			mockFunc: func(m *MockDynamoDB) {
				m.On("UpdateItem", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("dynamo error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDynamoDB)
			tt.mockFunc(mockDB)
			client := NewDynamoDBClient(mockDB, tt.tableName, NewNoopTelemetryClient())

			err := client.IncrementLinkCount(context.Background(), "123", 5)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockDB.AssertExpectations(t)
		})
	}
}
