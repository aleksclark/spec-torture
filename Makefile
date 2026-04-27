BINARY := spec-torture
MODULE := github.com/aleksclark/spec-torture
BUILD_DIR := bin

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

.PHONY: docker
docker:
	docker build -t spec-torture .
