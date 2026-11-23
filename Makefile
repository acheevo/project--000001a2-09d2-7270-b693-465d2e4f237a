.PHONY: help build run test test-unit test-integration clean docker-build docker-run docker-stop docker-clean lint fmt vet tidy deps ci dev-setup

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/api ./cmd/api

run: ## Run the application
	go run ./cmd/api

test: test-unit test-integration ## Run all tests (requires Docker)

test-unit: ## Run unit tests
	go test -race ./...

test-integration: ## Run integration tests
	go test -race -tags=integration ./tests/integration/...

clean: ## Clean build artifacts
	rm -rf bin/
	go clean

docker-build: ## Build Docker image
	docker build -t nimbus-rbac-stage .

docker-run: ## Run with Docker Compose
	docker compose up --build

docker-stop: ## Stop Docker Compose
	docker compose down

docker-clean: ## Clean Docker containers and volumes
	docker compose down -v
	docker rmi nimbus-rbac-stage 2>/dev/null || true

lint: ## Run golangci-lint
	golangci-lint run --build-tags=integration

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy go modules
	go mod tidy

deps: ## Download dependencies
	go mod download

ci: lint build test ## Run CI pipeline locally (requires Docker)

dev-setup: deps ## Download development dependencies

.PHONY: api-contract
api-contract: ## Export the typed OpenAPI contract without starting services
	@tmp=$$(mktemp docs/openapi.XXXXXX) || exit 1; \
	trap 'rm -f "$$tmp"' EXIT; \
	go run ./cmd/openapi > "$$tmp" && mv "$$tmp" docs/openapi.json
