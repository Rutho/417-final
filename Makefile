# Variables
APP_NAME=myapp           # Name of the built application
BUILD_DIR=build          # Directory for build artifacts
MAIN_FILE=main.go        # Entry point of the application
IMPORT_PATHS=./client/...  # Include directories for Go packages

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)

# Run the application
.PHONY: run
run: build
	$(BUILD_DIR)/$(APP_NAME)

# Format the Go code
.PHONY: fmt
fmt:
	go fmt ./...

# Lint the code (requires golangci-lint)
.PHONY: lint
lint:
	golangci-lint run

# Run tests
.PHONY: test
test:
	go test $(IMPORT_PATHS)

# Clean up build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
