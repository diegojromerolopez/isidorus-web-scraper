#!/bin/bash

# Create Queues
awslocal sqs create-queue --queue-name scraper-queue
awslocal sqs create-queue --queue-name image-queue
awslocal sqs create-queue --queue-name writer-queue

# Create DynamoDB Table for Job State
awslocal dynamodb create-table \
    --table-name scraping_jobs \
    --attribute-definitions AttributeName=job_id,AttributeType=S \
    --key-schema AttributeName=job_id,KeyType=HASH \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5

awslocal s3 mb s3://isidorus-images
