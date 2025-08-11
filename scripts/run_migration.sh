#!/bin/bash

set -e

echo "Running migrations..."

# Load environment variables (if not already loaded)
if [ -f /var/www/app/.env ]; then
  export $(grep -v '^#' /var/www/app/.env | xargs)
fi

# Construct full DB URL (adjust variables as needed)
DB_URL="postgres://${DB_USER}:${PG_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# Run migrations
migrate -path /app/migrations -database "$DB_URL" up

echo "Migrations completed."
