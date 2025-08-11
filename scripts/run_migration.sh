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

# Run migrations
migrate -path /app/migrations -database "$DB_URL" up

echo "Migrations completed."