.PHONY: run test fmt up down migrate

run:
	go run ./cmd/server

test:
	go test ./...

fmt:
	gofmt -w cmd internal

up:
	docker compose up --build

down:
	docker compose down

migrate:
	docker compose run --rm migrate
