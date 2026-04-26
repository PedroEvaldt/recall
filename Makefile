.PHONY: db-up db-down db-shell goose-up goose-down run-server build

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
	go build -o bin/cli ./cmd/cli
