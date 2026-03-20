.PHONY: run migrate

run:
	go run ./cmd/server

db-up:
	docker compose up db

test-db-up:
	docker compose up test_db

migrate:
	go run ./cmd/migrate
