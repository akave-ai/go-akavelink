# Project variables
PKG := ./...
CMD := ./cmd/server
PORT ?= 8080

.PHONY: help fmt fmt-check vet lint test race cover cover-html build run tidy install-tools ci

help:
	@echo "Common targets:"
	@echo "  fmt          - Format code with gofmt and goimports"
	@echo "  fmt-check    - Check formatting (fail if not formatted)"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run golangci-lint"
	@echo "  test         - Run unit tests"
	@echo "  race         - Run tests with race detector"
	@echo "  cover        - Run tests and report coverage summary"
	@echo "  cover-html   - Generate HTML coverage report"
	@echo "  build        - Build server"
	@echo "  run          - Run server (PORT=$(PORT))"
	@echo "  tidy         - go mod tidy"
	@echo "  install-tools- Install local dev tools (goimports, golangci-lint)"
	@echo "  ci           - fmt-check, vet, lint, test"

fmt:
	@echo "Formatting with gofmt..."
	gofmt -s -w .
	@echo "Fixing imports with goimports... (install with 'go install golang.org/x/tools/cmd/goimports@latest' if missing)"
	goimports -w .

fmt-check:
	@echo "Checking formatting..."
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

vet:
	go vet $(PKG)

lint:
	golangci-lint run

test:
	go test $(PKG) -count=1

race:
	go test $(PKG) -race -count=1

cover:
	go test $(PKG) -coverprofile=coverage.out -count=1
	@go tool cover -func coverage.out

cover-html: cover
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

build:
	go build $(CMD)

run:
	PORT=$(PORT) go run $(CMD)

tidy:
	go mod tidy

install-tools:
	@echo "Installing goimports..."
	GO111MODULE=on go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing golangci-lint v1.59.1..."
	GOBIN=$$(go env GOPATH)/bin; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$$GOBIN" v1.59.1; \
		echo "Installed golangci-lint to $$GOBIN";

ci: fmt-check vet lint test
