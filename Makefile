# Variables
APP_NAME ?= ledger-service
BUILD_DIR ?= bin
CMD_SERVER_PATH ?= ./cmd/server
CMD_MIGRATE_PATH ?= ./cmd/migrate
MIGRATION_PATH ?= ./internal/platform/database/postgres/migrations
GO ?= go

# Docker Variables
DOCKER_COMPOSE_PROD_FILE ?= docker/production/docker-compose.yaml
DOCKER_PROD_FILE ?= docker/production/Dockerfile
DOCKER_COMPOSE_DEV_FILE ?= docker/development/docker-compose.yaml
DOCKER_DEV_FILE ?= docker/development/Dockerfile
DOCKER_IMAGE ?= $(APP_NAME)
DOCKER_TAG ?= latest
PORT ?= 8080

.PHONY: help all build run test test-short lint fmt tidy clean migrate migration migrate-up migrate-down sqlc gen-mocks docker-build docker-build-dev docker-run docker-run-dev docker-compose-up docker-compose-down docker-clean

# Default target when running just 'make'
.DEFAULT_GOAL := help

## help: Display this list of available commands and descriptions
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

###########
## GO
###########

## build: Compile the binary into the bin directory
build:
	@mkdir -p $(BUILD_DIR)
	@echo "Building $(APP_NAME)..."
	@$(GO) build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(CMD_SERVER_PATH)

## run: Build and execute the application locally
run: build
	@echo "Running $(APP_NAME)..."
	@./$(BUILD_DIR)/$(APP_NAME)

## test: Run unit tests with race detection and coverage
test:
	@echo "Running tests..."
	@$(GO) test -v -race -cover ./...

## test-short: Run short unit tests only
test-short:
	@echo "Running short tests..."
	@$(GO) test -v -short ./...

## lint: Run golangci-lint static analysis
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

## fmt: Run go fmt and go vet across all packages
fmt:
	@echo "Formatting and vetting code..."
	@$(GO) fmt ./...
	@$(GO) vet ./...

## tidy: Cleanup and verify go.mod dependencies
tidy:
	@echo "Tidying go module dependencies..."
	@$(GO) mod tidy
	@$(GO) mod verify

## clean: Remove build artifacts and temporary files
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@$(GO) clean


	
############
## SQLC
############

## migration: Create Migrations
migration: 
	@migrate create -ext sql -dir $(MIGRATION_PATH) $(filter-out $@,$(MAKECMDGOALS))


## migrate-up: Run up Migrations
migrate-up: 
	@$(GO) run $(CMD_MIGRATE_PATH) up

## migrate-down: Run down Migrations
migrate-down: 
	@$(GO) run $(CMD_MIGRATE_PATH) down


## sqlc: Generate SQL code with sqlc
sqlc: 
	@sqlc generate
	@$(GO) generate ./...


## gen-mocks: Generate Mocks
gen-mocks:
	@$(GO) generate ./...



###########
## DOCKER
###########

## docker-build: Build production Docker image
docker-build:
	@echo "Building production Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker build \
		-f $(DOCKER_PROD_FILE) \
		--build-arg CMD_SERVER_PATH=$(CMD_SERVER_PATH) \
		--build-arg PORT=$(PORT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) .

## docker-build-dev: Build development Docker image
docker-build-dev:
	@echo "Building development Docker image $(DOCKER_IMAGE):dev..."
	@docker build \
		-f $(DOCKER_DEV_FILE) \
		-t $(DOCKER_IMAGE):dev .

## docker-run: Run production Docker container locally
docker-run: docker-build
	@echo "Running production container $(DOCKER_IMAGE):$(DOCKER_TAG) on port $(PORT)..."
	@docker run --rm -p $(PORT):$(PORT) -e PORT=$(PORT) --name $(APP_NAME) --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

## docker-run-dev: Run dev Docker container with live reload & volume mount
docker-run-dev: docker-build-dev
	@echo "Running development container with volume mount..."
	@docker run --rm -p $(PORT):$(PORT) -e PORT=$(PORT) -v $(CURDIR):/app --name $(APP_NAME)-dev --env-file .env $(DOCKER_IMAGE):dev

## docker-compose-up: Spin up development services via docker-compose
docker-compose-up:
	@echo "Starting Development Docker Compose services..."
	@docker compose -f $(DOCKER_COMPOSE_DEV_FILE) up -d --build

## docker-compose-down: Stop development docker-compose services
docker-compose-down:
	@echo "Stopping Development Docker Compose services..."
	@docker compose -f $(DOCKER_COMPOSE_DEV_FILE) down -v

## docker-compose-prod-up: Spin up production services via docker-compose
docker-compose-prod-up:
	@echo "Starting Production Docker Compose services..."
	@docker compose -f $(DOCKER_COMPOSE_PROD_FILE) up -d --build

## docker-compose-prod-down: Stop production docker-compose services
docker-compose-prod-down:
	@echo "Stopping Production Docker Compose services..."
	@docker compose -f $(DOCKER_COMPOSE_PROD_FILE) down -v

## docker-clean: Remove unused Docker containers and dangling images
docker-clean:
	@echo "Cleaning dangling Docker resources..."
	@docker system prune -f