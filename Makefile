.PHONY: fmt test test-race vet build migrate migrate-down seed web-install web-test web-build run compose-up compose-down

fmt:
	gofmt -w cmd internal tests

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

build:
	go build ./...

migrate:
	go run ./cmd/migrate

migrate-down:
	go run ./cmd/migrate down

seed: migrate

web-install:
	cd web && npm ci

web-test:
	cd web && npm test

web-build:
	cd web && npm run build

run:
	go run ./cmd/server

compose-up:
	docker compose up --build

compose-down:
	docker compose down
