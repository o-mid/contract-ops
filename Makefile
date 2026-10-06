GOLANGCI_LINT_VERSION := 2.13.2
BIN_DIR := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint

.PHONY: test test-api test-web lint up down

test: test-api test-web

test-api:
	cd api && go test ./...

test-web:
	npm run check

lint: $(GOLANGCI_LINT)
	cd api && $(GOLANGCI_LINT) run ./...
	cd api && $(GOLANGCI_LINT) fmt --diff

$(GOLANGCI_LINT):
	mkdir -p $(BIN_DIR)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/v$(GOLANGCI_LINT_VERSION)/install.sh | sh -s -- -b $(BIN_DIR) v$(GOLANGCI_LINT_VERSION)

up:
	docker compose up --build -d

down:
	docker compose down
