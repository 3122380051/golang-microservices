APP_NAME=golang-microservices
GOFILES=$(shell go list ./...)

.PHONY: setup run test lint build migrate-up migrate-down migrate-status

setup:
	chmod +x scripts/*.sh
	cp -n .env.example .env 2>/dev/null || true
	docker compose up -d --build

run:
	go run ./cmd/api-gateway

test:
	go test ./...

lint:
	go vet ./...

build:
	go build -o bin/api-gateway ./cmd/api-gateway

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status
