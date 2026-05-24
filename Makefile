.PHONY: generate migrate run build test

GOCACHE ?= $(CURDIR)/.gocache

generate:
	sqlc generate

migrate:
	GOCACHE=$(GOCACHE) go run . --migrate-only

run:
	GOCACHE=$(GOCACHE) go run .

build:
	GOCACHE=$(GOCACHE) go build .

test:
	GOCACHE=$(GOCACHE) go test ./...
