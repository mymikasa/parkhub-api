SHELL := /bin/bash

GO ?= go
BUF ?= buf
WIRE ?= wire
GOOSE ?= goose
GOLANGCI_LINT ?= golangci-lint
DOCKER_COMPOSE ?= docker compose

BIN_DIR := bin
MONOLITH_BIN := $(BIN_DIR)/parkhub

.PHONY: help proto-gen proto-lint proto-breaking proto-descriptor lint lint-tenant test test-integration build-monolith docker-build wire migrate generate-keys docker-up docker-down docker-ps clean

help:
	@echo "Available targets:"
	@echo "  proto-gen         Generate protobuf code with buf"
	@echo "  proto-lint        Run buf lint"
	@echo "  proto-breaking    Run buf breaking check against local main branch"
	@echo "  lint              Run golangci-lint"
	@echo "  lint-tenant       Run tenant linter if configured"
	@echo "  test              Run unit tests"
	@echo "  test-integration  Run integration tests"
	@echo "  build-monolith    Build cmd/monolith to bin/parkhub"
	@echo "  docker-build      Build monolith Docker image"
	@echo "  generate-keys     Generate RSA key pair for JWT signing"
	@echo "  wire              Generate Wire DI code if configured"
	@echo "  migrate           Run goose migrations if configured"
	@echo "  docker-up         Start docker compose services"
	@echo "  docker-down       Stop docker compose services"
	@echo "  docker-ps         Show docker compose service status"
	@echo "  clean             Remove build artifacts"

proto-gen:
	@command -v $(BUF) >/dev/null 2>&1 || { echo "buf is required but not installed"; exit 1; }
	$(BUF) generate
	@$(MAKE) proto-descriptor

proto-descriptor:
	@command -v $(BUF) >/dev/null 2>&1 || { echo "buf is required but not installed"; exit 1; }
	$(BUF) build -o configs/apisix/proto-descriptor.pb

proto-lint:
	@command -v $(BUF) >/dev/null 2>&1 || { echo "buf is required but not installed"; exit 1; }
	$(BUF) lint

proto-breaking:
	@command -v $(BUF) >/dev/null 2>&1 || { echo "buf is required but not installed"; exit 1; }
	$(BUF) breaking --against '.git#branch=main'

lint:
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || { echo "golangci-lint is required but not installed"; exit 1; }
	$(GOLANGCI_LINT) run ./...

lint-tenant:
	@if [ ! -d tools/tenantlint ]; then \
		echo "tools/tenantlint is not present yet"; \
		exit 1; \
	fi
	@echo "tenant linter is not wired yet"
	@exit 1

test:
	$(GO) test ./...

test-integration:
	$(GO) test -tags=integration ./...

build-monolith:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(MONOLITH_BIN) ./cmd/monolith

docker-build:
	docker build -t parkhub-monolith .

wire:
	@command -v $(WIRE) >/dev/null 2>&1 || { echo "wire is required but not installed"; exit 1; }
	@if [ ! -f cmd/monolith/wire.go ]; then \
		echo "cmd/monolith/wire.go is not present yet"; \
		exit 1; \
	fi
	$(WIRE) ./cmd/monolith/...

migrate:
	@command -v $(GOOSE) >/dev/null 2>&1 || { echo "goose is required but not installed"; exit 1; }
	@if [ ! -d migrations ]; then \
		echo "migrations directory is not present yet"; \
		exit 1; \
	fi
	@if [ -z "$$DATABASE_URL" ]; then \
		echo "DATABASE_URL is required, for example: parkhub:parkhub@tcp(localhost:3306)/parkhub?charset=utf8mb4&parseTime=True&loc=Local"; \
		exit 1; \
	fi
	$(GOOSE) -dir migrations mysql "$$DATABASE_URL" up

generate-keys:
	@if [ ! -f configs/keys/jwt_private.pem ]; then \
		mkdir -p configs/keys && \
		openssl genpkey -algorithm RSA -out configs/keys/jwt_private.pem -pkeyopt rsa_keygen_bits:2048 && \
		openssl rsa -in configs/keys/jwt_private.pem -pubout -out configs/keys/jwt_public.pem && \
		chmod 644 configs/keys/jwt_private.pem configs/keys/jwt_public.pem && \
		echo "RSA keys generated in configs/keys/"; \
	else \
		echo "RSA keys already exist in configs/keys/"; \
	fi

docker-up: generate-keys
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-ps:
	$(DOCKER_COMPOSE) ps

clean:
	rm -rf $(BIN_DIR)
