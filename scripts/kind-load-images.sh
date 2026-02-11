#!/bin/bash
set -e

echo "🐳 Retagging images..."
docker tag isidorus-web-scraper-api:latest api:latest
docker tag isidorus-web-scraper-auth-admin:latest auth-admin:latest
docker tag isidorus-web-scraper-frontend:latest frontend:latest
docker tag isidorus-web-scraper-scraper-worker:latest scraper-worker:latest
docker tag isidorus-web-scraper-writer-worker:latest writer-worker:latest
docker tag isidorus-web-scraper-indexer-worker:latest indexer-worker:latest
docker tag isidorus-web-scraper-image-extractor-worker:latest image-extractor-worker:latest
docker tag isidorus-web-scraper-image-explainer-worker:latest image-explainer-worker:latest
docker tag isidorus-web-scraper-page-summarizer-worker:latest page-summarizer-worker:latest
docker tag isidorus-web-scraper-deletion-worker:latest deletion-worker:latest

echo "✅ Images retagged."

echo "🚚 Loading images into Kind (sequentially to save memory)..."

IMAGES=(
  api:latest
  auth-admin:latest
  frontend:latest
  scraper-worker:latest
  writer-worker:latest
  indexer-worker:latest
  image-extractor-worker:latest
  image-explainer-worker:latest
  page-summarizer-worker:latest
  deletion-worker:latest
)

CLUSTER_NAME="${1:-isidorus}"

for img in "${IMAGES[@]}"; do
  echo "📦 Loading $img into cluster $CLUSTER_NAME..."
  kind load docker-image "$img" --name "$CLUSTER_NAME"
done

echo "🎉 All images loaded into Kind cluster!"
