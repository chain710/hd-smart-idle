GO ?= go
GOFLAGS ?= -mod=readonly
BIN_DIR ?= bin
BINARY ?= $(BIN_DIR)/hd-smart-idle
GOLANGCI_LINT_VERSION ?= v2.6.0

GOLANGCI_LINT_BIN := $(BIN_DIR)/golangci-lint
MOCKERY_BIN := $(BIN_DIR)/mockery

.PHONY: all build test lint clean

all: build

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

generate: $(MOCKERY_BIN)
	$(MOCKERY_BIN)

build: $(BIN_DIR)
	$(GO) build $(GOFLAGS) -o $(BINARY) github.com/chain710/hd-smart-idle

test:
	$(GO) test $(GOFLAGS) -v -race -count 1 ./...

lint: $(GOLANGCI_LINT_BIN)
	$(GOLANGCI_LINT_BIN) run

lintfix: $(GOLANGCI_LINT_BIN)
	$(GOLANGCI_LINT_BIN) run --fix

$(GOLANGCI_LINT_BIN): | $(BIN_DIR)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(BIN_DIR) $(GOLANGCI_LINT_VERSION)

$(MOCKERY_BIN): | $(BIN_DIR)
	curl -sSfL https://github.com/vektra/mockery/releases/download/v3.5.5/mockery_3.5.5_Linux_x86_64.tar.gz | tar -xz -C $(BIN_DIR)

clean:
	rm -rf $(BIN_DIR)
