#!/bin/bash
# scripts/dockerhub-cleanup.sh
set -e

if [ -z "$DOCKERHUB_USERNAME" ]; then
    echo "Error: DOCKERHUB_USERNAME is not set"
    exit 1
fi

if [ -z "$DOCKERHUB_TOKEN" ]; then
    echo "Error: DOCKERHUB_TOKEN is not set"
    exit 1
fi

if [ -z "$BRANCH" ]; then
    echo "Error: BRANCH is not set"
    exit 1
fi

# main and master branches should never be cleaned up this way
if [ "$BRANCH" == "main" ] || [ "$BRANCH" == "master" ]; then
    echo "Cleanup skipped for protected branch: $BRANCH"
    exit 0
fi

# Slugify the branch name (must match the logic in dockerhub-push.yml)
BRANCH_SLUG=$(echo "$BRANCH" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/^-//;s/-$//')

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

echo "🔑 Obtaining Docker Hub authentication token..."
AUTH_TOKEN=$(curl -s -H "Content-Type: application/json" -X POST \
  -d '{"username": "'"$DOCKERHUB_USERNAME"'", "password": "'"$DOCKERHUB_TOKEN"'"}' \
  https://hub.docker.com/v2/users/login/ | jq -r .token)

if [ "$AUTH_TOKEN" == "null" ] || [ -z "$AUTH_TOKEN" ]; then
    echo "❌ Error: Failed to obtain Docker Hub token. Check credentials."
    exit 1
fi

for svc in "${SERVICES[@]}"; do
    REPO="isidorus-$svc-$BRANCH_SLUG"
    echo "🧹 Deleting repository: $DOCKERHUB_USERNAME/$REPO..."
    
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" \
      -X DELETE \
      -H "Authorization: JWT $AUTH_TOKEN" \
      "https://hub.docker.com/v2/repositories/$DOCKERHUB_USERNAME/$REPO/")
    
    if [ "$RESPONSE" == "204" ]; then
        echo "✅ Successfully deleted $REPO"
    elif [ "$RESPONSE" == "404" ]; then
        echo "ℹ️ Repository $REPO not found (might have been already deleted or never pushed)"
    else
        echo "❌ Failed to delete $REPO (HTTP status: $RESPONSE)"
    fi
done

echo "🎉 Docker Hub cleanup for branch '$BRANCH' complete!"
