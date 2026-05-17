#!/usr/bin/env sh
set -eu

if [ ! -f .env ] && [ -f .env.example ]; then
  cp .env.example .env
fi

docker compose up -d --build
