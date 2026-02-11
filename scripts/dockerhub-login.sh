#!/bin/bash
# scripts/dockerhub-login.sh
set -e

if [ -z "$DOCKERHUB_USERNAME" ]; then
    echo "Error: DOCKERHUB_USERNAME is not set"
    exit 1
fi

if [ -z "$DOCKERHUB_TOKEN" ]; then
    echo "Error: DOCKERHUB_TOKEN is not set"
    exit 1
fi

echo "Logging into Docker Hub as $DOCKERHUB_USERNAME..."
echo "$DOCKERHUB_TOKEN" | docker login -u "$DOCKERHUB_USERNAME" --password-stdin
