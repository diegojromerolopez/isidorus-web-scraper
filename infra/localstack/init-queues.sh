#!/bin/bash

awslocal sqs create-queue --queue-name scraper-queue
awslocal sqs create-queue --queue-name image-queue
awslocal sqs create-queue --queue-name writer-queue
awslocal s3 mb s3://nube2e-images
