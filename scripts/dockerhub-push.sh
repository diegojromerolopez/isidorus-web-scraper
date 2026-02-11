#!/bin/bash
# scripts/dockerhub-push.sh
set -e


if [ -z "$DOCKER_USER" ]; then
    echo "Error: DOCKER_USER is not set. Please provide it as an environment variable or ensure \$USER is defined."
    exit 1
fi

# Read version from .semver if TAG is not provided
if [ -z "$TAG" ]; then
    if [ -f ".semver" ]; then
        TAG=$(cat .semver | tr -d '[:space:]')
        echo "Reading version from .semver: $TAG"
    else
        TAG="latest"
    fi
fi

# List of services to build and push
SERVICES=(
  "api"
  "auth-admin"
  "frontend"
  "scraper-worker"
  "writer-worker"
  "indexer-worker"
  "image-extractor-worker"
  "image-explainer-worker"
  "page-summarizer-worker"
  "deletion-worker"
)

# Map services to their directory names (where they differ)
declare -A DIRS
DIRS["api"]="api"
DIRS["auth-admin"]="auth_admin"
DIRS["frontend"]="frontend"
DIRS["scraper-worker"]="workers/scraper"
DIRS["writer-worker"]="workers/writer"
DIRS["indexer-worker"]="workers/indexer"
DIRS["image-extractor-worker"]="workers/image_extractor"
DIRS["image-explainer-worker"]="workers/image_explainer"
DIRS["page-summarizer-worker"]="workers/page_summarizer"
DIRS["deletion-worker"]="workers/deletion"

echo "Using Docker username: $DOCKER_USER"
echo "Using tag: $TAG"

for svc in "${SERVICES[@]}"; do
    echo "---------------------------------------------------"
    echo "Building and pushing isidorus-$svc..."
    dir=${DIRS[$svc]}
    
    if [ -z "$dir" ]; then
        echo "Error: Directory for service $svc not found"
        exit 1
    fi

    IMAGE_NAME="isidorus-$svc"
    if [ -n "$BRANCH" ] && [ "$BRANCH" != "main" ] && [ "$BRANCH" != "master" ]; then
        IMAGE_NAME="${IMAGE_NAME}-${BRANCH}"
    fi

    # Determine build context. Python services need 'shared/' from the root.
    # Go, Frontend, and Auth services are designed for local context.
    CONTEXT="."
    case "$svc" in
        api|image-explainer-worker|page-summarizer-worker|deletion-worker)
            CONTEXT="."
            ;;
        *)
            CONTEXT="$dir"
            ;;
    esac

    echo "Context: $CONTEXT"
    docker build -t "$DOCKER_USER/$IMAGE_NAME:$TAG" -f "$dir/Dockerfile" "$CONTEXT"
    docker push "$DOCKER_USER/$IMAGE_NAME:$TAG"
done

echo "---------------------------------------------------"
echo "All images pushed successfully!"
