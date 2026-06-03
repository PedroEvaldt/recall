# Development task runner for local database, migrations, builds, and install.
.PHONY: db-up db-down db-shell goose-up goose-down run-server build install test-unit test-all test-coverage

# Default local database URL used by goose targets.
DB_URL?=postgres://recall:recall@localhost:5433/recall?sslmode=disable

db-up:
	docker compose up -d

db-down:
	docker compose down

db-shell:
	docker compose exec postgres psql -U recall -d recall

goose-up:
	GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DB_URL)" goose -dir internal/storage/schemas up

goose-down:
	GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DB_URL)" goose -dir internal/storage/schemas down

goose-status:
	GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DB_URL)" goose -dir internal/storage/schemas status

run-server:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server
	go build -o bin/recall ./cmd/recall

install:
	go install ./cmd/recall

# Tests shortcuts

test-unit:
	go test ./...

test-all:
	go test -tags=integratinon -count=1 ./...

test-coverage:
	go test -tags=integration -coverprofile=/tmp/cov.out -coverpkg=./internal/handlers ./internal/handlers/ && go tool cover -html=/tmp/cov.out
