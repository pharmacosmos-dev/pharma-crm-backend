#!/bin/bash

set -e

echo "Running migrations..."

# Load environment variables safely
if [ -f /var/www/app/.env ]; then
  set -a
  . /var/www/app/.env
  set +a
fi

# Construct full DB URL
DB_URL="postgres://${PG_USER}:${PG_PASS}@${PG_HOST}:${PG_PORT}/${PG_DB}?sslmode=disable"

# Optional: Reset dirty flag and force to latest migration version
LATEST_VERSION=$(ls /app/migrations | grep -E '^[0-9]+' | sort -n | tail -1 | cut -d'_' -f1)
echo "Forcing migration version to $LATEST_VERSION"
migrate -path /app/migrations -database "$DB_URL" force "$LATEST_VERSION"

# Run migrations
migrate -path /app/migrations -database "$DB_URL" up

echo "Migrations completed."