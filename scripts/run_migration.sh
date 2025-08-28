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

# Agar migration "dirty" bo‘lsa, tozalab olish
if migrate -path /app/migrations -database "$DB_URL" version 2>&1 | grep -q "dirty"; then
  echo "Database is dirty, forcing to clean state..."
  VERSION=$(migrate -path /app/migrations -database "$DB_URL" version | awk '{print $1}')
  migrate -path /app/migrations -database "$DB_URL" force "$VERSION"
fi

# Run migrations
migrate -path /app/migrations -database "$DB_URL" up

echo "Migrations completed."