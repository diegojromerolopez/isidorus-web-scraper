#!/bin/bash
# scripts/k8s-cloud-prepare.sh
set -e

REGISTRY_URL="${1}"
if [ -z "$REGISTRY_URL" ]; then
    echo "Usage: $0 <your-registry-url-prefix>"
    echo "Example: $0 1234567890.dkr.ecr.us-east-1.amazonaws.com/isidorus"
    exit 1
fi

APPS_DIR="k8s/apps"
INFRA_DIR="k8s/infra"

echo "☁️ Preparing manifests for cloud deployment (Registry: $REGISTRY_URL)..."

# 1. Update image references to use the cloud registry
echo "🖼️ Updating image names..."
find "$APPS_DIR" -name "*.yaml" -exec sed -i '' "s|image: docker.io/library/|image: $REGISTRY_URL-|g" {} +

# 2. Change Frontend service to LoadBalancer
echo "🌐 Updating frontend service to LoadBalancer..."
sed -i '' 's/type: ClusterIP/type: LoadBalancer/' "$APPS_DIR/frontend-service.yaml"

# 3. Warning about storage classes
echo "⚠️  Note: You must manually update StatefulSets in $INFRA_DIR to use your cloud's StorageClass."
echo "    Look for 'storageClassName' in volumeClaimTemplates."

echo "✅ Manifests prepared for cloud push."
echo "👉 Next steps: Use 'make dockerhub-push' (with updated scripts) to push to your registry."
