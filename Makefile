.PHONY: generate refresh-currencies migrate run build test

GOCACHE ?= $(CURDIR)/.gocache

generate:
	sqlc generate

refresh-currencies:
	GOCACHE=$(GOCACHE) go run ./cmd/refresh-currencies

migrate:
	GOCACHE=$(GOCACHE) go run . --migrate-only

run:
	GOCACHE=$(GOCACHE) go run .

build:
	GOCACHE=$(GOCACHE) go build .

test:
	GOCACHE=$(GOCACHE) go test ./...
