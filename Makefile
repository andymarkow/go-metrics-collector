# Usage:
# make        		# run default command

# To check entire script:
# cat -e -t -v Makefile

.EXPORT_ALL_VARIABLES:

LOG_LEVEL=debug
RESTORE=false
STORE_INTERVAL=10
FILE_STORAGE_PATH=./metrics-db.json
KEY=secretkey
# DATABASE_DSN=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
USE_GRPC=true

.PHONY: all
all: fmt tidy test lint

.PHONY: fmt
fmt:
	go fmt ./...
	$(HOME)/go/bin/goimports -l -w --local "github.com/andymarkow/go-metrics-collector" .

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: run-server
run-server:
	go run ./cmd/server

.PHONY: run-agent
run-agent:
	ADDRESS=localhost:50051 go run ./cmd/agent

.PHONY: run-postgres
run-postgres:
	docker-compose up postgres pgadmin

.PHONY: stop-postgres
stop-postgres:
	docker-compose down postgres pgadmin

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	$(HOME)/go/bin/staticcheck -fail "" ./...
	docker run --rm --name golangci-lint -v `pwd`:/workspace -w /workspace \
		golangci/golangci-lint:latest-alpine golangci-lint run --issues-exit-code 1

.PHONY: staticlint
staticlint:
	go run ./cmd/staticlint ./...

.PHONY: test
test:
	go clean -testcache
	go test -race -v ./...

.PHONY: coverage
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

.PHONY: pprof
pprof:
	### CPU profile
	# go tool pprof -http=":9090" -seconds=30 http://localhost:8080/debug/pprof/profile
	### Memory profile
	# curl -sK -v http://localhost:8080/debug/pprof/heap > heap.out
	# go tool pprof -http=":9090" -seconds=30 http://localhost:8080/debug/pprof/heap
	# go tool pprof -http=":9090" -seconds=30 heap.out

.PHONY: gen-proto-v1
gen-proto-v1:
	protoc \
		--proto_path=internal/grpc/proto/metric/v1 \
		--go_out=internal/grpc/api/metric/v1 \
		--go_opt=paths=source_relative \
		--go-grpc_out=internal/grpc/api/metric/v1 \
		--go-grpc_opt=paths=source_relative \
		internal/grpc/proto/metric/v1/*.proto
