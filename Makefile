include tools.mk

# Default environment variables (can be overridden from environment)
HOST ?= 0.0.0.0
PORT ?= 9871
BASE_URL ?= http://localhost
PODCAST_TITLE ?= 'my podcast'
PODCAST_DESCRIPTION ?= 'my awesome podcast'
PODCAST_AUTHOR ?= 'authour'
DB_PATH ?= ./db/podcast.db

generate: tools-install
	$(TOOLS_BIN)/oapi-codegen -config docs/schema/v1/config.yaml docs/schema/v1/podcast.yaml

sqlc: tools-install
	$(TOOLS_BIN)/sqlc generate

migrate-up:
	$(TOOLS_BIN)/migrate -database "sqlite3://$(DB_PATH)" -path sql/migrations up

migrate-down:
	$(TOOLS_BIN)/migrate -database "sqlite3://$(DB_PATH)" -path sql/migrations down

build:
	go build -o bin/server ./cmd/server

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	$(TOOLS_BIN)/golangci-lint run

fmt:
	$(TOOLS_BIN)/gofumpt -w .

dev: tools-install
	DB_PATH=$(DB_PATH) HOST=$(HOST) PORT=$(PORT) BASE_URL=$(BASE_URL) PODCAST_TITLE=$(PODCAST_TITLE) PODCAST_DESCRIPTION=$(PODCAST_DESCRIPTION) PODCAST_AUTHOR=$(PODCAST_AUTHOR) $(TOOLS_BIN)/air -c .air.toml

run:
	DB_PATH=$(DB_PATH) HOST=$(HOST) PORT=$(PORT) BASE_URL=$(BASE_URL) PODCAST_TITLE=$(PODCAST_TITLE) PODCAST_DESCRIPTION=$(PODCAST_DESCRIPTION) PODCAST_AUTHOR=$(PODCAST_AUTHOR) go run ./cmd/server
