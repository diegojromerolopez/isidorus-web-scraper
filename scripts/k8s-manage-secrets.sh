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
  generate-app-secrets)
    # Secrets for the application (AWS credentials, shared keys, etc.)
    # We construct DATABASE_URL dynamically so it's ready for the app
    # Default host is 'postgres.isidorus.svc.cluster.local'
    DB_HOST="postgres.isidorus.svc.cluster.local"
    DB_PORT="5432"
    # Ensure variables have defaults if not set in environment (for safety, though generate_secret checks too)
    P_USER="${POSTGRES_USER:-postgres}"
    P_PASS="${POSTGRES_PASSWORD:-postgres}"
    P_DB="${POSTGRES_DB:-isidorus}"
    
    # Export for generate_secret to pick up
    export DATABASE_URL="postgres://$P_USER:$P_PASS@$DB_HOST:$DB_PORT/$P_DB"
    
    generate_secret "k8s-secrets" "AWS_ACCESS_KEY_ID" "AWS_SECRET_ACCESS_KEY" "POSTGRES_PASSWORD" "POSTGRES_USER" "POSTGRES_DB" "DATABASE_URL"
    ;;
  apply)
    # Apply Postgres secrets
    kubectl create secret generic postgres-secret \
      --namespace=$NS \
      --from-literal=POSTGRES_USER="${POSTGRES_USER:-postgres}" \
      --from-literal=POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}" \
      --from-literal=POSTGRES_DB="${POSTGRES_DB:-isidorus}" \
      --dry-run=client -o yaml | kubectl apply -f -

    # Apply App secrets
    # Construct DATABASE_URL for application
    DB_HOST="postgres.isidorus.svc.cluster.local"
    DB_PORT="5432"
    P_USER="${POSTGRES_USER:-postgres}"
    P_PASS="${POSTGRES_PASSWORD:-postgres}"
    P_DB="${POSTGRES_DB:-isidorus}"
    DATABASE_URL="postgres://$P_USER:$P_PASS@$DB_HOST:$DB_PORT/$P_DB"

    kubectl create secret generic k8s-secrets \
      --namespace=$NS \
      --from-literal=AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID}" \
      --from-literal=AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}" \
      --from-literal=POSTGRES_PASSWORD="${P_PASS}" \
      --from-literal=POSTGRES_USER="${P_USER}" \
      --from-literal=POSTGRES_DB="${P_DB}" \
      --from-literal=DATABASE_URL="${DATABASE_URL}" \
      --dry-run=client -o yaml | kubectl apply -f -
    ;;
  *)
    echo "Usage: $0 {generate-postgres|generate-app-secrets|apply}"
    exit 1
    ;;
esac
