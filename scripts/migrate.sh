#!/usr/bin/env sh
set -eu

action="${1:-up}"
DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/golang_microservices?sslmode=disable}"

case "$action" in
  up|down|status)
    go run ./cmd/migrate "$action"
    ;;
  *)
    echo "usage: $0 {up|down|status}" >&2
    exit 1
    ;;
esac
