#!/bin/bash
set -e

echo "Waiting for Minio and DynamoDB to be ready..."
sleep 5

# Configure AWS CLI for local use (matching Minio/LocalStack creds)
# Credentials are passed via environment variables
export AWS_DEFAULT_REGION=us-east-1

echo "Creating S3 Buckets..."
# Create bucket if it doesn't exist
aws --endpoint-url http://minio:9000 s3 mb s3://isidorus-images || echo "Bucket already exists"

echo "Creating DynamoDB Tables (ScyllaDB Alternator)..."
# Create table if it doesn't exist
aws --endpoint-url http://scylla:8000 dynamodb create-table \
    --table-name scraping_jobs \
    --attribute-definitions AttributeName=scraping_id,AttributeType=S \
    --key-schema AttributeName=scraping_id,KeyType=HASH \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 || echo "Table already exists"

echo "Initialization complete!"
