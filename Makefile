# Usage:
# make        		# run default command

# To check entire script:
# cat -e -t -v Makefile

.EXPORT_ALL_VARIABLES:

FILE_STORAGE_PATH=metrics-db.json
STORE_INTERVAL=10
DATABASE_DSN=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable

.PHONY: all

all: fmt tidy test lint

fmt:
	go fmt ./...

tidy:
	go mod tidy

run-server:
	go run ./cmd/server

run-agent:
	go run ./cmd/agent

run-postgres:
	docker-compose up postgres pgadmin

stop-postgres:
	docker-compose down postgres pgadmin

lint:
	docker run --rm --name golangci-lint -v `pwd`:/workspace -w /workspace \
		golangci/golangci-lint:latest-alpine golangci-lint run --issues-exit-code 1

test:
	go clean -testcache
	go test -race -v ./...

coverage:
	go clean -testcache
	go test -v -cover -coverprofile=.coverage.cov ./...
	go tool cover -func=.coverage.cov
	go tool cover -html=.coverage.cov
	rm .coverage.cov
