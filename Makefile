.PHONY: run migrate

run:
	go run ./cmd/relay

migrate:
	go run ./cmd/migrate
