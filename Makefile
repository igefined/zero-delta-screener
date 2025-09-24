.PHONY: help build run test clean docker-build docker-run docker-stop fmt vendor lint

# Variables
APP_NAME := screener
DOCKER_IMAGE := zero-delta-screener
DOCKER_TAG := latest
GOFLAGS := -v
LDFLAGS := -ldflags="-w -s"
MAIN_PATH := main.go
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
GOLANGCI_LINT_VERSION := v1.62.2

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

## help: Display this help message
help:
	@echo "$(GREEN)Available targets:$(NC)"
	@grep -E '^##' Makefile | sed 's/## //'

## build: Build the application binary
build:
	@echo "$(YELLOW)Building $(APP_NAME)...$(NC)"
	@mkdir -p $(BIN_DIR)
	@go build $(GOFLAGS) $(LDFLAGS) -o $(BIN_PATH) $(MAIN_PATH)
	@echo "$(GREEN)Build complete: $(BIN_PATH)$(NC)"

## run: Run the application locally
run:
	@echo "$(YELLOW)Running $(APP_NAME)...$(NC)"
	@go run $(MAIN_PATH)

## test: Run tests
test:
	@echo "$(YELLOW)Running tests...$(NC)"
	@go test $(GOFLAGS) ./...
	@echo "$(GREEN)Tests complete$(NC)"

## test-coverage: Run tests with coverage
test-coverage:
	@echo "$(YELLOW)Running tests with coverage...$(NC)"
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

## fmt: Format code
fmt:
	@echo "$(YELLOW)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)Formatting complete$(NC)"

## vet: Run go vet
vet:
	@echo "$(YELLOW)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)Vet complete$(NC)"

## mod-tidy: Clean up module dependencies
mod-tidy:
	@echo "$(YELLOW)Tidying modules...$(NC)"
	@go mod tidy
	@echo "$(GREEN)Modules tidied$(NC)"

## mod-download: Download module dependencies
mod-download:
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	@go mod download
	@echo "$(GREEN)Dependencies downloaded$(NC)"

## vendor: Create vendor directory
vendor:
	@echo "$(YELLOW)Creating vendor directory...$(NC)"
	@go mod vendor
	@echo "$(GREEN)Vendor directory created$(NC)"

## clean: Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean complete$(NC)"

## docker-build: Build Docker image
docker-build:
	@echo "$(YELLOW)Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)...$(NC)"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

## docker-run: Run application in Docker
docker-run:
	@echo "$(YELLOW)Running Docker container...$(NC)"
	@docker run --rm \
		--name $(APP_NAME) \
		--env-file .env \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

## docker-run-detached: Run application in Docker (detached)
docker-run-detached:
	@echo "$(YELLOW)Running Docker container (detached)...$(NC)"
	@docker run -d \
		--name $(APP_NAME) \
		--env-file .env \
		--restart unless-stopped \
		$(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(GREEN)Container started: $(APP_NAME)$(NC)"

## docker-stop: Stop Docker container
docker-stop:
	@echo "$(YELLOW)Stopping Docker container...$(NC)"
	@docker stop $(APP_NAME) || true
	@docker rm $(APP_NAME) || true
	@echo "$(GREEN)Container stopped$(NC)"

## docker-logs: View Docker container logs
docker-logs:
	@docker logs -f $(APP_NAME)

## docker-push: Push Docker image to registry
docker-push:
	@echo "$(YELLOW)Pushing Docker image...$(NC)"
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(GREEN)Docker image pushed$(NC)"

## docker-compose-up: Start services with docker-compose
docker-compose-up:
	@echo "$(YELLOW)Starting services with docker-compose...$(NC)"
	@docker-compose up -d
	@echo "$(GREEN)Services started$(NC)"

## docker-compose-down: Stop services with docker-compose
docker-compose-down:
	@echo "$(YELLOW)Stopping services with docker-compose...$(NC)"
	@docker-compose down
	@echo "$(GREEN)Services stopped$(NC)"

## docker-compose-logs: View docker-compose logs
docker-compose-logs:
	@docker-compose logs -f

## install-tools: Install development tools
install-tools:
	@echo "$(YELLOW)Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(GREEN)Tools installed$(NC)"

## lint: Run golangci-lint
lint:
	@echo "$(YELLOW)Running linter...$(NC)"
	@golangci-lint run ./...
	@echo "$(GREEN)Linting complete$(NC)"

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "$(GREEN)All checks passed$(NC)"

## ci: Run CI pipeline locally
ci: clean mod-tidy check build
	@echo "$(GREEN)CI pipeline complete$(NC)"

# Default target
.DEFAULT_GOAL := help