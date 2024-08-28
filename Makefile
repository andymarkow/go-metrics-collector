# Usage:
# make        		# run default command

# To check entire script:
# cat -e -t -v Makefile

.EXPORT_ALL_VARIABLES:

LOG_LEVEL=debug
RESTORE=false
STORE_INTERVAL=10
FILE_STORAGE_PATH=
KEY=secretkey
# DATABASE_DSN=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable

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

vet:
	go vet ./...

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

# benchmark:
# 	go test -v -bench .
# 	go test -bench . -benchmem
# 	go test -v -bench -benchmem -benchtime=10s .

pprof:
	### CPU profile
	# go tool pprof -http=":9090" -seconds=30 http://localhost:8080/debug/pprof/profile
	### Memory profile
	# curl -sK -v http://localhost:8080/debug/pprof/heap > heap.out
	# go tool pprof -http=":9090" -seconds=30 http://localhost:8080/debug/pprof/heap
	# go tool pprof -http=":9090" -seconds=30 heap.out
