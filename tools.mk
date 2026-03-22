# Tooling configuration
TOOLS_DIR := tools
TOOLS_BIN := $(abspath $(TOOLS_DIR)/bin)
GO ?= go

# Versions
GOLANGCI_LINT_VERSION ?= v2.11.3
GOFUMPT_VERSION ?= v0.9.2
AIR_VERSION ?= v1.64.5
OAPI_CODEGEN_VERSION ?= v2.6.0
SQLC_VERSION ?= v1.30.0

# Binaries
GOLANGCI_LINT := $(TOOLS_BIN)/golangci-lint
GOFUMPT := $(TOOLS_BIN)/gofumpt
AIR := $(TOOLS_BIN)/air
OAPI_CODEGEN := $(TOOLS_BIN)/oapi-codegen
SQLC := $(TOOLS_BIN)/sqlc

# Build variables
GIT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X 'main.GitSHA=$(GIT_SHA)' -X 'main.BuildTime=$(BUILD_TIME)'

.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  tools-install   Install required build tools to $(TOOLS_BIN)"
	@echo "  tools-clean     Remove installed build tools from $(TOOLS_BIN)"
	@echo "  generate        Generate Go code from OpenAPI specification"
	@echo "  test            Run tests"
	@echo "  lint            Run linter"
	@echo "  build           Build server binary"
	@echo "  debug-build     Build server binary with debug symbols"
	@echo "  format          Format source code"
	@echo "  clean           Clean build artifacts and tools"

.PHONY: tools-dir
tools-dir:
	mkdir -p $(TOOLS_BIN)

.PHONY: tools-install
tools-install: tools-dir $(GOLANGCI_LINT) $(GOFUMPT) $(AIR) $(OAPI_CODEGEN) $(SQLC)

$(GOLANGCI_LINT):
	GOBIN=$(TOOLS_BIN) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(GOFUMPT):
	GOBIN=$(TOOLS_BIN) $(GO) install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)

$(AIR):
	GOBIN=$(TOOLS_BIN) $(GO) install github.com/air-verse/air@$(AIR_VERSION)

$(OAPI_CODEGEN):
	GOBIN=$(TOOLS_BIN) $(GO) install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)


$(SQLC):
	GOBIN=$(TOOLS_BIN) $(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)

.PHONY: tools-clean
tools-clean:
	rm -rf $(TOOLS_BIN)

.PHONY: generate
# generate is defined in Makefile (calls oapi-codegen and sqlc)
