# Usage:
# make        		# run default command

# To check entire script:
# cat -e -t -v Makefile

.EXPORT_ALL_VARIABLES:

.PHONY: all lint validate docs build cloudbuild

all: fmt tidy

fmt:
	go fmt

tidy:
	go mod tidy

run-server:
	go run ./cmd/server

run-agent:
	go run ./cmd/agent

test:
	go test -v -cover -coverprofile=profile.cov ./...
	go tool cover -func profile.cov
	rm profile.cov
