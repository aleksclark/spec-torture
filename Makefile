BINARY := spec-torture
MODULE := github.com/aleksclark/spec-torture
BUILD_DIR := bin
GOBIN := $(shell go env GOPATH)/bin

.PHONY: all build clean test lint run validate

all: build

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/spec-torture

clean:
	rm -rf $(BUILD_DIR) spec-torture.db

test:
	go test ./...

lint:
	go vet ./...

validate:
	$(BUILD_DIR)/$(BINARY) validate specs/mcp/spec.yaml

run:
	$(BUILD_DIR)/$(BINARY) run specs/mcp/spec.yaml --runtime test --image alpine:latest

tidy:
	go mod tidy

# ---------------------------------------------------------------------------
# ARP reference implementation (gRPC server + client libraries)
# ---------------------------------------------------------------------------

.PHONY: arp-tools proto arp arp-server arp-conformance arp-test arp-run

# Install the protobuf code generators used by `make proto`.
arp-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Regenerate gen/arp/v1 from proto/arp/v1 (requires buf + arp-tools on PATH).
proto:
	PATH="$(GOBIN):$$PATH" buf generate

arp-server:
	go build -o $(BUILD_DIR)/arp-server ./cmd/arp-server

arp-conformance:
	go build -o $(BUILD_DIR)/arp-conformance ./cmd/arp-conformance

arp: arp-server arp-conformance

# Run the in-process Go conformance suite.
arp-test:
	go test ./arp/...

# Build + boot the seeded reference server and run both conformance suites,
# writing reports to reports/arp/.
arp-run: arp
	agents/arp-reference/run.sh reports/arp

.PHONY: docker
docker:
	docker build -t spec-torture .
