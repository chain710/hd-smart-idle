GO ?= go
GOFLAGS ?= -mod=readonly
BIN_DIR ?= bin
BINARY ?= $(BIN_DIR)/hd-smart-idle
GOLANGCI_LINT_VERSION ?= v2.6.0

GOLANGCI_LINT_BIN := $(BIN_DIR)/golangci-lint

.PHONY: all build test lint clean

all: build

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	$(GO) build $(GOFLAGS) -o $(BINARY) github.com/example/hd-smart-idle

test:
	$(GO) test $(GOFLAGS) -v -race -count 1 ./...

lint: $(GOLANGCI_LINT_BIN)
	$(GOLANGCI_LINT_BIN) run

lintfix: $(GOLANGCI_LINT_BIN)
	$(GOLANGCI_LINT_BIN) run --fix

$(GOLANGCI_LINT_BIN): | $(BIN_DIR)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(BIN_DIR) $(GOLANGCI_LINT_VERSION)

clean:
	rm -rf $(BIN_DIR)
