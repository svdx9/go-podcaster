GO ?= go
# Default environment variables (can be overridden from environment)
HOST ?= 0.0.0.0
PORT ?= 9871
BASE_URL ?= http://localhost
PODCAST_TITLE ?= 'my podcast'
PODCAST_DESCRIPTION ?= 'my awesome podcast'
PODCAST_AUTHOR ?= 'authour'
DB_PATH ?= ./db/podcast.db

include tools.mk

generate: tools-install
	$(TOOLS_BIN)/oapi-codegen -config docs/schema/v1/config.yaml docs/schema/v1/podcast.yaml

sqlc: tools-install
	$(TOOLS_BIN)/sqlc generate

build:
	$(GO) build -o bin/server ./cmd/server

docker-build:
	docker build -t go-podcaster:local .

test:
	$(GO) test -race -coverprofile=coverage.out ./...

lint:
	$(TOOLS_BIN)/golangci-lint run

fmt:
	$(TOOLS_BIN)/gofumpt -w .

dev: tools-install
	DB_PATH=$(DB_PATH) HOST=$(HOST) PORT=$(PORT) BASE_URL=$(BASE_URL) PODCAST_TITLE=$(PODCAST_TITLE) PODCAST_DESCRIPTION=$(PODCAST_DESCRIPTION) PODCAST_AUTHOR=$(PODCAST_AUTHOR) $(TOOLS_BIN)/air -c .air.toml

run:
	DB_PATH=$(DB_PATH) HOST=$(HOST) PORT=$(PORT) BASE_URL=$(BASE_URL) PODCAST_TITLE=$(PODCAST_TITLE) PODCAST_DESCRIPTION=$(PODCAST_DESCRIPTION) PODCAST_AUTHOR=$(PODCAST_AUTHOR) $(GO) run ./cmd/server
