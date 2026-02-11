#!/bin/bash
# scripts/k8s-setup.sh
set -e

CLUSTER_NAME="isidorus"
NS="isidorus"

echo "🚀 Starting Isidorus Kubernetes setup..."

# 1. Check for tools
for tool in docker kind kubectl; do
    if ! command -v $tool &> /dev/null; then
        echo "❌ Error: $tool is not installed."
        exit 1
    fi
done

# 2. Ensure Docker is running
if ! docker info &> /dev/null; then
    echo "🐳 Starting Docker Desktop..."
    open -a Docker
    echo "⏳ Waiting for Docker to be ready..."
    while ! docker info &> /dev/null; do
        sleep 5
    done
fi
echo "✅ Docker is running."

# 3. Create Kind Cluster
if ! kind get clusters | grep -q "^$CLUSTER_NAME$"; then
    echo "🏗️ Creating Kind cluster '$CLUSTER_NAME'..."
    kind create cluster --name "$CLUSTER_NAME"
else
    echo "✅ Kind cluster '$CLUSTER_NAME' already exists."
fi

# 4. Build Docker Images
echo "📦 Building Docker images..."
docker compose -f docker-compose.base.yml -f docker-compose.yml build

# 5. Load Images into Kind
echo "🚚 Loading images into Kind $CLUSTER_NAME..."
bash scripts/kind-load-images.sh "$CLUSTER_NAME"

# 6. Apply Manifests
echo "📄 Applying Kubernetes manifests..."

# Install KEDA (using the official manifest for easy installation)
echo "⚡ Installing KEDA Operator..."
kubectl apply -f https://github.com/kedacore/keda/releases/download/v2.13.0/keda-2.13.0.yaml

kubectl apply -f k8s/base/namespace.yaml

# Apply secrets (using defaults if variables are missing)
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-test}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-test}"
export POSTGRES_USER="${POSTGRES_USER:-postgres}"
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
export POSTGRES_DB="${POSTGRES_DB:-isidorus}"
bash scripts/k8s-manage-secrets.sh apply

# Apply infrastructure and applications
kubectl apply -R -f k8s/infra/
kubectl apply -R -f k8s/apps/

echo "⏳ Waiting for pods to be ready (this may take a few minutes)..."
kubectl wait --for=condition=Ready pods --all -n "$NS" --timeout=300s || echo "⚠️ Some pods are taking longer to start."

echo "--------------------------------------------------"
echo "🎉 Isidorus is now running on Kubernetes!"
echo "--------------------------------------------------"
echo "Frontend: http://localhost:3000"
echo "API:      http://localhost:8000"
echo ""
echo "To access the application, run:"
echo "👉 make k8s-port-forward"
echo "--------------------------------------------------"
