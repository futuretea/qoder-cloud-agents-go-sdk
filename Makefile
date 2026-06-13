# Qoder Cloud Agents Go SDK Makefile

.PHONY: all build test lint fmt vet tidy clean help ci

# Default target
all: lint test build

# Build the project (verification only for library)
build:
	@echo "Building..."
	go build -v ./...

# Run all tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.txt -o coverage.html

# Run golangci-lint
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Tidy and download dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy
	go mod download

# Verify dependencies
verify:
	@echo "Verifying dependencies..."
	go mod verify

# Clean generated files
clean:
	@echo "Cleaning..."
	rm -f coverage.txt coverage.html
	go clean -cache

# Run all checks (useful for CI)
ci: tidy verify fmt vet lint test build

# Show help
help:
	@echo "Available targets:"
	@echo "  make build          - Build the project"
	@echo "  make test           - Run all tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make fmt            - Format code with go fmt"
	@echo "  make vet            - Run go vet"
	@echo "  make tidy           - Tidy and download dependencies"
	@echo "  make verify         - Verify dependencies"
	@echo "  make clean          - Clean generated files"
	@echo "  make ci             - Run all checks (CI pipeline)"
	@echo "  make all            - Run lint, test, and build"
