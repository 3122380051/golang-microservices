#!/usr/bin/env sh
set -eu

BACKUP_DIR="${BACKUP_DIR:-backups}"
DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/golang_microservices?sslmode=disable}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_FILE="$BACKUP_DIR/golang_microservices_$TIMESTAMP.sql"

mkdir -p "$BACKUP_DIR"
pg_dump "$DATABASE_URL" > "$BACKUP_FILE"

echo "backup created: $BACKUP_FILE"
