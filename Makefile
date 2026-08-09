.PHONY: all build test lint hygiene ci clean install-local setup-local run

BINARY := ldash
CMD := ./cmd/ldash
BUILD_DIR := bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD)

test:
	go test ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

hygiene:
	@chmod +x scripts/ci-hygiene.sh
	@./scripts/ci-hygiene.sh

ci:
	go vet ./...
	$(MAKE) test
	$(MAKE) hygiene
	@if command -v golangci-lint >/dev/null 2>&1; then \
		$(MAKE) lint; \
	else \
		echo "make ci: golangci-lint not installed; skipping lint (required in GitHub Actions)"; \
	fi
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "make ci: govulncheck not installed; skipping vuln (required in GitHub Actions)"; \
	fi

clean:
	rm -rf $(BUILD_DIR) dist/

install-local: build
	install -d "$$HOME/.local/bin"
	install -m 755 $(BUILD_DIR)/$(BINARY) "$$HOME/.local/bin/$(BINARY)"

setup-local:
	@chmod +x scripts/setup-local.sh
	@./scripts/setup-local.sh

run: build
	./$(BUILD_DIR)/$(BINARY)
