#!/bin/bash
# scripts/k8s-port-forward.sh
set -e

NS="isidorus"

echo "🌐 Port-forwarding Frontend to http://localhost:3000..."
echo "🚀 Port-forwarding API to http://localhost:8000..."

# Run port-forwards in the background
kubectl port-forward svc/frontend 3000:3000 -n $NS &
kubectl port-forward svc/api 8000:8000 -n $NS &

# Wait for background processes
wait
