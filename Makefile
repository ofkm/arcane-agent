.PHONY: build run clean test

BINARY_NAME=arcane-agent
BUILD_DIR=bin

build:
    go build -o $(BUILD_DIR)/$(BINARY_NAME) ./

run: build
    ./$(BUILD_DIR)/$(BINARY_NAME)

clean:
    rm -rf $(BUILD_DIR)

test:
    go test ./...

install-deps:
    go mod tidy
    go mod download

dev:
    go run main.go