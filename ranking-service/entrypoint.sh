#!/bin/sh
set -e

echo "Waiting for database to be ready..."
sleep 5

echo "Running database migrations..."
/app/migrate

echo "Starting ranking service..."
exec "$@"
