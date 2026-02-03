#!/bin/bash

# Create Queues
awslocal sqs create-queue --queue-name scraper-queue
awslocal sqs create-queue --queue-name image-extractor-queue
awslocal sqs create-queue --queue-name writer-queue
awslocal sqs create-queue --queue-name page-summarizer-queue

# Create DynamoDB Table for Job State
awslocal dynamodb create-table \
    --table-name scraping_jobs \
    --attribute-definitions AttributeName=scraping_id,AttributeType=S \
    --key-schema AttributeName=scraping_id,KeyType=HASH \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5

awslocal s3 mb s3://isidorus-images
