# ---- Config ----
SHELL := /bin/bash
GO ?= go
GOLANGCI_LINT_VERSION ?= v1.64.2
GOLANGCI_LINT ?= $(shell command -v golangci-lint 2>/dev/null || echo "$(GOPATH)/bin/golangci-lint")

# ---- Helpers ----
.PHONY: help
help: ## Show available targets
	@awk 'BEGIN {FS":.*##"; printf "\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# ---- Tooling ----
.PHONY: tools
tools: ## Ensure linters/tools are installed
	@test -x "$(GOLANGCI_LINT)" || { \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
		  | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	}

# ---- Lint ----
.PHONY: lint
lint: tools ## Run linters
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: tools ## Run linters with auto-fix (where supported)
	$(GOLANGCI_LINT) run --fix

# ---- Go basics ----
.PHONY: tidy
tidy: ## go mod tidy + vendor
	$(GO) mod tidy
	@if [ -f go.mod ]; then $(GO) mod verify; fi

.PHONY: test
test: ## Run tests
	$(GO) test ./...

.PHONY: ci
ci: tidy lint test ## Run full CI suite (tidy, lint, test)
	@echo "✅ CI checks passed"
