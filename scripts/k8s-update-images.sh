#!/bin/bash
# scripts/k8s-update-images.sh
set -e

DOCKER_USER=${DOCKER_USER:-""}
TAG=${TAG:-""}

if [ -z "$DOCKER_USER" ]; then
    echo "Error: DOCKER_USER is not set."
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

echo "Updating Kubernetes manifests to use Docker Hub user: $DOCKER_USER, Tag: $TAG"

# Services list from dockerhub-push.sh
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

# Loop through all application manifests
for file in k8s/apps/*-deployment.yaml; do
    echo "Processing $file..."
    for svc in "${SERVICES[@]}"; do
        # Replace 'image: service' with 'image: user/isidorus-service:tag'
        # We use a strict match for the service name to avoid partial replacements
        sed -i "" "s/image: $svc$/image: $DOCKER_USER\/isidorus-$svc:$TAG/" "$file"
    done
done

echo "K8s image paths updated successfully!"
