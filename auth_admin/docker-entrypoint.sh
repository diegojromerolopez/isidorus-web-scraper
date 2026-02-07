#!/bin/sh

set -e

echo "Running migrations..."
python manage.py migrate --noinput

if [ "$SEED_DATA" = "true" ]; then
    echo "Seeding test data..."
    python manage.py setup_test_data
fi

echo "Starting server..."
exec "$@"
