.PHONY: all build run clean proto test lint

all: proto build

proto:
	buf generate

build:
	go build -o bin/ad-server ./cmd/ad-server
	go build -o bin/ad-manager ./cmd/ad-manager
	go build -o bin/ad-syncer ./cmd/ad-syncer
	go build -o bin/file-gateway ./cmd/file-gateway

run-server:
	go run ./cmd/ad-server

run-manager:
	go run ./cmd/ad-manager

run-syncer:
	go run ./cmd/ad-syncer

run-gateway:
	go run ./cmd/file-gateway

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

clean:
	rm -rf bin/
