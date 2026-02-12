#!/bin/bash
set -e

ROLE="standby"
if [[ $(hostname) == "postgres-0" ]]; then
    ROLE="primary"
fi

echo "🚀 Starting Postgres $(hostname) as $ROLE..."

if [[ "$ROLE" == "primary" ]]; then
    # Primary setup is handled by standard docker-entrypoint.sh
    # We just need to make sure replication is allowed
    # This script will be called by docker-entrypoint-initdb.d/
    psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "CREATE USER replicator WITH REPLICATION ENCRYPTED PASSWORD 'replicator';"
    echo "host replication replicator 0.0.0.0/0 md5" >> "$PGDATA/pg_hba.conf"
else
    # Standby setup
    # Wait for primary to be ready
    until pg_isready -h postgres-0.postgres.isidorus.svc.cluster.local -p 5432; do
        echo "⏳ Waiting for primary (postgres-0) to be ready..."
        sleep 2
    done

    # If data directory is empty, run pg_basebackup
    if [ ! -s "$PGDATA/PG_VERSION" ]; then
        echo "📂 Data directory is empty, running pg_basebackup..."
        rm -rf "$PGDATA"/*
        export PGPASSWORD='replicator'
        pg_basebackup -h postgres-0.postgres.isidorus.svc.cluster.local -D "$PGDATA" -U replicator -vP -R -X stream
        echo "✅ pg_basebackup completed."
    fi
fi

# The standard docker-entrypoint.sh will take over
exec docker-entrypoint.sh "$@"
