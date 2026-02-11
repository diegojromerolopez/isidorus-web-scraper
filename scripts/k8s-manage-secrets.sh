#!/bin/bash
# scripts/k8s-manage-secrets.sh
set -e

# Load .env if it exists
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

SECRETS_DIR="k8s/base/secrets"
NS="isidorus"

# Function to generate secret from environment variables
generate_secret() {
  local secret_name=$1
  shift
  local keys=("$@")

  echo "---"
  echo "apiVersion: v1"
  echo "kind: Secret"
  echo "metadata:"
  echo "  name: $secret_name"
  echo "  namespace: $NS"
  echo "type: Opaque"
  echo "stringData:"

  for key in "${keys[@]}"; do
    val="${!key}"
    if [ -z "$val" ]; then
      echo "Error: Environment variable $key is not set" >&2
      exit 1
    fi
    echo "  $key: $val"
  done
}

# Example: generate_secret "postgres-secret" "POSTGRES_USER" "POSTGRES_PASSWORD" "POSTGRES_DB" > k8s/base/secrets/postgres-secret.yaml

case "$1" in
  generate-postgres)
    generate_secret "postgres-secret" "POSTGRES_USER" "POSTGRES_PASSWORD" "POSTGRES_DB"
    ;;
  apply)
    # Applying secrets directly using from-literal is often cleaner than YAML files
    # but generating YAML allows for better auditing/review if needed.
    # Here we'll just apply them.
    kubectl create secret generic postgres-secret \
      --namespace=$NS \
      --from-literal=POSTGRES_USER="${POSTGRES_USER:-postgres}" \
      --from-literal=POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}" \
      --from-literal=POSTGRES_DB="${POSTGRES_DB:-isidorus}" \
      --dry-run=client -o yaml | kubectl apply -f -
    ;;
  *)
    echo "Usage: $0 {generate-postgres|apply}"
    exit 1
    ;;
esac
