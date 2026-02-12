#!/bin/bash
set -e

ROLE="standby"
if [[ $(hostname) == "postgres-0" ]]; then
    ROLE="primary"
fi

echo "🚀 Starting Postgres $(hostname) as $ROLE..."

if [[ "$ROLE" == "primary" ]]; then
    # Primary setup
    (
        echo "Primary background task started. Waiting for pg_hba.conf..."
        until [ -f "$PGDATA/pg_hba.conf" ]; do
            sleep 1
        done
        if ! grep -q "replicator" "$PGDATA/pg_hba.conf"; then
            echo "host replication replicator 0.0.0.0/0 md5" >> "$PGDATA/pg_hba.conf"
            echo "Added replicator to pg_hba.conf"
            # Wait for postgres to be ready to reload
            until pg_isready -h localhost -p 5432; do
                sleep 1
            done
            psql -U "$POSTGRES_USER" -c "SELECT pg_reload_conf();"
            echo "Reloaded postgres configuration"
        fi
    ) &
else
    # Standby setup
    until pg_isready -h postgres-0.postgres.isidorus.svc.cluster.local -p 5432; do
        echo "⏳ Waiting for primary (postgres-0) to be ready..."
        sleep 2
    done

    if [ ! -s "$PGDATA/PG_VERSION" ]; then
        echo "📂 Data directory is empty, running pg_basebackup..."
        rm -rf "${PGDATA:?}"/*
        export PGPASSWORD='replicator'
        pg_basebackup -h postgres-0.postgres.isidorus.svc.cluster.local -D "$PGDATA" -U replicator -vP -R -X stream
        echo "✅ pg_basebackup completed."
    fi
fi
exec docker-entrypoint.sh "$@"
