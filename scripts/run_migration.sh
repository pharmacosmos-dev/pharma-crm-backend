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
VERSION_OUTPUT=$(migrate -path /app/migrations -database "$DB_URL" version 2>&1 || true)
if printf '%s' "$VERSION_OUTPUT" | grep -q "dirty"; then
  DIRTY_VERSION=$(printf '%s\n' "$VERSION_OUTPUT" | sed -nE 's/^([0-9]+).*/\1/p')
  PREVIOUS_VERSION=$(find /app/migrations -maxdepth 1 -type f -name '*.up.sql' \
    | sed -nE 's#^.*/0*([0-9]+)_.*\.up\.sql$#\1#p' \
    | awk -v version="$DIRTY_VERSION" '$1 < version { candidate = $1 } END { print candidate }')

  if [ -z "$DIRTY_VERSION" ] || [ -z "$PREVIOUS_VERSION" ]; then
    echo "Could not safely recover dirty migration state: $VERSION_OUTPUT" >&2
    exit 1
  fi

  echo "Database is dirty at version $DIRTY_VERSION; resetting to $PREVIOUS_VERSION before retrying."
  migrate -path /app/migrations -database "$DB_URL" force "$PREVIOUS_VERSION"
fi

# Run migrations
migrate -path /app/migrations -database "$DB_URL" up

echo "Migrations completed."
