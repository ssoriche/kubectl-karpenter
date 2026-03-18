# kubectl-karpenter Makefile

BINARY_NAME := kubectl-karpenter
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: help setup build build-all test lint install clean release

help: ## Show this help message
	@echo "kubectl-karpenter - Karpenter NodePool resource utilization"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Set up git hooks for development
	git config core.hooksPath .githooks
	@echo "Git hooks configured."

build: ## Build for current platform
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/kubectl-karpenter

build-all: ## Build for all platforms
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)_linux_amd64 ./cmd/kubectl-karpenter
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)_linux_arm64 ./cmd/kubectl-karpenter
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)_darwin_amd64 ./cmd/kubectl-karpenter
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)_darwin_arm64 ./cmd/kubectl-karpenter
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)_windows_amd64.exe ./cmd/kubectl-karpenter

test: ## Run tests
	go test -v -race ./...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run ./...

install: build ## Install to GOPATH/bin
	cp bin/$(BINARY_NAME) $(GOPATH)/bin/

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

release: ## Create release with goreleaser (requires goreleaser)
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed"; exit 1; }
	goreleaser release --clean

release-snapshot: ## Create snapshot release (for testing)
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed"; exit 1; }
	goreleaser release --snapshot --clean

tidy: ## Tidy go modules
	go mod tidy

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

check: fmt vet lint test ## Run all checks
