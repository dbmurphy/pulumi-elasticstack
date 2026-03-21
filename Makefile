PROJECT_NAME := elasticstack
PROVIDER_BIN := bin/pulumi-resource-$(PROJECT_NAME)
GO_MODULE := github.com/dbmurphy/pulumi-elasticstack/provider
VERSION := 0.1.0

.PHONY: help build clean clean-sdk test test-race lint vet vulncheck fmt-check tidy tidy-check ci schema gen-sdk-nodejs gen-sdk-python gen-sdk-go gen-sdk-dotnet gen-sdk hooks unhooks

.DEFAULT_GOAL := help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: tidy ## Build the provider binary
	cd provider && go build -o ../$(PROVIDER_BIN) -ldflags "-X github.com/dbmurphy/pulumi-elasticstack/provider/pkg/provider.Version=$(VERSION)" ./cmd/pulumi-resource-$(PROJECT_NAME)

clean: clean-sdk ## Remove all build artifacts and generated SDKs
	rm -rf bin/
	rm -f schema.json

clean-sdk: ## Remove generated SDK directories
	rm -rf sdk/nodejs sdk/python sdk/go sdk/dotnet

test: ## Run tests (matches CI: race detector + coverage)
	cd provider && go test ./... -v -count=1 -race -coverprofile=coverage.out
	@echo "Coverage report: provider/coverage.out"

test-short: ## Run tests without race detector (faster local iteration)
	cd provider && go test ./... -count=1

lint: ## Run golangci-lint (matches CI)
	cd provider && golangci-lint config verify
	cd provider && golangci-lint run ./...

vet: ## Run go vet
	cd provider && go vet ./...

vulncheck: ## Run govulncheck for known vulnerabilities (matches CI)
	cd provider && govulncheck ./...

fmt-check: ## Check formatting (gofmt + gci) without modifying files
	@echo "Checking gofmt..."
	@cd provider && test -z "$$(gofmt -l .)" || { echo "Files need gofmt:"; gofmt -l .; exit 1; }
	@echo "Checking gci import order..."
	@cd provider && test -z "$$(gci diff -s standard -s blank -s default -s 'prefix(github.com/dbmurphy/pulumi-elasticstack)' ./pkg/ ./cmd/ 2>/dev/null)" || { echo "Files need gci:"; gci diff -s standard -s blank -s default -s 'prefix(github.com/dbmurphy/pulumi-elasticstack)' ./pkg/ ./cmd/; exit 1; }
	@echo "Formatting OK"

fmt: ## Auto-fix formatting (gofmt + gci)
	cd provider && gofmt -w .
	cd provider && gci write -s standard -s blank -s default -s 'prefix(github.com/dbmurphy/pulumi-elasticstack)' ./pkg/ ./cmd/

tidy: ## Run go mod tidy
	cd provider && go mod tidy

tidy-check: tidy ## Verify go.mod/go.sum are tidy (fails if tidy changes anything)
	@if [ -n "$$(git diff -- provider/go.mod provider/go.sum)" ]; then \
		echo "go.mod or go.sum is not tidy. Run 'make tidy' and commit."; \
		git diff -- provider/go.mod provider/go.sum; \
		exit 1; \
	fi

ci: lint vet test build tidy-check ## Run all CI checks locally (lint + vet + test + build + tidy)
	@echo ""
	@echo "All CI checks passed."

ci-full: lint vet vulncheck test build tidy-check ## Run all CI checks including vulncheck
	@echo ""
	@echo "All CI checks (including vulncheck) passed."

schema: build ## Generate Pulumi schema JSON
	pulumi package get-schema ./$(PROVIDER_BIN) > schema.json

gen-sdk-nodejs: schema ## Generate Node.js SDK
	pulumi package gen-sdk ./$(PROVIDER_BIN) --language nodejs -o sdk/nodejs

gen-sdk-python: schema ## Generate Python SDK
	pulumi package gen-sdk ./$(PROVIDER_BIN) --language python -o sdk/python

gen-sdk-go: schema ## Generate Go SDK
	pulumi package gen-sdk ./$(PROVIDER_BIN) --language go -o sdk/go

gen-sdk-dotnet: schema ## Generate .NET SDK
	pulumi package gen-sdk ./$(PROVIDER_BIN) --language dotnet -o sdk/dotnet

gen-sdk: gen-sdk-nodejs gen-sdk-python gen-sdk-go gen-sdk-dotnet ## Generate all SDKs

hooks: ## Install git pre-commit hook
	@ln -sf ../../scripts/pre-commit .git/hooks/pre-commit
	@echo "Pre-commit hook installed. Use 'git commit --no-verify' to skip."

unhooks: ## Remove git pre-commit hook
	@rm -f .git/hooks/pre-commit
	@echo "Pre-commit hook removed."
