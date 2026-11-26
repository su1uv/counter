DATABASE_URL ?= postgres://counter:counter@localhost:5432/counter?sslmode=disable
TEST_DATABASE_URL ?= postgres://counter:counter@localhost:5432/counter_test?sslmode=disable
export DATABASE_URL

.PHONY: build run test test-integration cover db-up migrate-up migrate-down migrate-status migrate-new sqlc sqlc-check

build:
	go build -o bin/migrate ./cmd/migrate
	go build -o bin/counter ./cmd/counter

run:
	go run ./cmd/counter

test:
	go test ./...

# -p 1: packages share one test database, so they must not run in parallel.
test-integration:
	TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test -p 1 ./...

cover:
	TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test -p 1 -coverprofile=coverage.out ./internal/config ./internal/server ./internal/migrate ./cmd/...
	go tool cover -func=coverage.out

db-up:
	docker compose up -d --wait postgres

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

migrate-new:
	goose -s -dir db/migrations create $(name) sql

sqlc:
	sqlc generate

sqlc-check:
	sqlc diff
