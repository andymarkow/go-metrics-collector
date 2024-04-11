# Usage:
# make        		# run default command

# To check entire script:
# cat -e -t -v Makefile

.EXPORT_ALL_VARIABLES:

.PHONY: all

all: fmt tidy

fmt:
	go fmt ./...

tidy:
	go mod tidy

run-server:
	go run ./cmd/server

run-agent:
	go run ./cmd/agent

lint:
	docker run --rm --name golangci-lint -v `pwd`:/workspace -w /workspace golangci/golangci-lint:latest-alpine golangci-lint run --issues-exit-code 1

test:
	go clean -testcache
	go test -race -v ./...

coverage:
	go clean -testcache
	go test -v -cover -coverprofile=.coverage.cov ./...
	go tool cover -func=.coverage.cov
	go tool cover -html=.coverage.cov
	rm .coverage.cov
