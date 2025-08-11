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
DB_URL="postgres://${DB_USER}:${PG_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# Run migrations
migrate -path /app/migrations -database "$DB_URL" up

echo "Migrations completed."