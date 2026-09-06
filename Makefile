SHELL := /bin/sh

GO_MOD_CACHE ?= /private/tmp/personal-tools-go-modcache
DIST_DIR ?= dist
BIN_DIR ?= $(CURDIR)/bin
GOLANGCI_LINT_VERSION := v2.12.2
GOLANGCI_LINT ?= $(BIN_DIR)/golangci-lint
AUTHORIZER_DIR := services/authorizer
MCP_DIR := services/mcp
CREDENTIALS_DIR := services/credentials
CREDENTIAL_AGENT_DIR := clients/credential-agent

.PHONY: test test-authorizer test-mcp test-credentials test-credential-agent vet lint install-lint race check build build-authorizer build-mcp build-credentials build-credential-agent clean

test: test-authorizer test-mcp test-credentials test-credential-agent

test-authorizer:
	cd $(AUTHORIZER_DIR) && GOCACHE=/private/tmp/personal-tools-authorizer-go-cache GOMODCACHE=$(GO_MOD_CACHE) go test ./...

test-mcp:
	cd $(MCP_DIR) && GOCACHE=/private/tmp/personal-tools-mcp-go-cache GOMODCACHE=$(GO_MOD_CACHE) go test ./...

test-credentials:
	cd $(CREDENTIALS_DIR) && GOCACHE=/private/tmp/personal-tools-credentials-go-cache GOMODCACHE=$(GO_MOD_CACHE) go test ./...

test-credential-agent:
	cd $(CREDENTIAL_AGENT_DIR) && GOCACHE=/private/tmp/personal-tools-credential-agent-go-cache GOMODCACHE=$(GO_MOD_CACHE) go test ./...

vet:
	cd $(AUTHORIZER_DIR) && GOCACHE=/private/tmp/personal-tools-authorizer-go-cache GOMODCACHE=$(GO_MOD_CACHE) go vet ./...
	cd $(MCP_DIR) && GOCACHE=/private/tmp/personal-tools-mcp-go-cache GOMODCACHE=$(GO_MOD_CACHE) go vet ./...
	cd $(CREDENTIALS_DIR) && GOCACHE=/private/tmp/personal-tools-credentials-go-cache GOMODCACHE=$(GO_MOD_CACHE) go vet ./...
	cd $(CREDENTIAL_AGENT_DIR) && GOCACHE=/private/tmp/personal-tools-credential-agent-go-cache GOMODCACHE=$(GO_MOD_CACHE) go vet ./...

install-lint:
	mkdir -p $(BIN_DIR)
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(BIN_DIR) $(GOLANGCI_LINT_VERSION)

lint:
	@test -x $(GOLANGCI_LINT) || { echo "golangci-lint missing; run 'make install-lint'" >&2; exit 1; }
	cd $(AUTHORIZER_DIR) && GOCACHE=/private/tmp/personal-tools-authorizer-go-cache GOMODCACHE=$(GO_MOD_CACHE) GOLANGCI_LINT_CACHE=/private/tmp/personal-tools-golangci-cache $(GOLANGCI_LINT) run -c ../../.golangci.yml ./...
	cd $(MCP_DIR) && GOCACHE=/private/tmp/personal-tools-mcp-go-cache GOMODCACHE=$(GO_MOD_CACHE) GOLANGCI_LINT_CACHE=/private/tmp/personal-tools-golangci-cache $(GOLANGCI_LINT) run -c ../../.golangci.yml ./...
	cd $(CREDENTIALS_DIR) && GOCACHE=/private/tmp/personal-tools-credentials-go-cache GOMODCACHE=$(GO_MOD_CACHE) GOLANGCI_LINT_CACHE=/private/tmp/personal-tools-golangci-cache $(GOLANGCI_LINT) run -c ../../.golangci.yml ./...
	cd $(CREDENTIAL_AGENT_DIR) && GOCACHE=/private/tmp/personal-tools-credential-agent-go-cache GOMODCACHE=$(GO_MOD_CACHE) GOLANGCI_LINT_CACHE=/private/tmp/personal-tools-golangci-cache $(GOLANGCI_LINT) run -c ../../.golangci.yml ./...

race:
	cd $(AUTHORIZER_DIR) && GOCACHE=/private/tmp/personal-tools-authorizer-go-cache GOMODCACHE=$(GO_MOD_CACHE) go test -race ./...
	cd $(MCP_DIR) && GOCACHE=/private/tmp/personal-tools-mcp-go-cache GOMODCACHE=$(GO_MOD_CACHE) go test -race ./...
	cd $(CREDENTIALS_DIR) && GOCACHE=/private/tmp/personal-tools-credentials-go-cache GOMODCACHE=$(GO_MOD_CACHE) go test -race ./...
	cd $(CREDENTIAL_AGENT_DIR) && GOCACHE=/private/tmp/personal-tools-credential-agent-go-cache GOMODCACHE=$(GO_MOD_CACHE) go test -race ./...

check: test vet lint race

build: build-authorizer build-mcp build-credentials build-credential-agent

build-authorizer:
	mkdir -p $(DIST_DIR)/authorizer
	cd $(AUTHORIZER_DIR) && GOCACHE=/private/tmp/personal-tools-authorizer-go-cache GOMODCACHE=$(GO_MOD_CACHE) CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ../../$(DIST_DIR)/authorizer/bootstrap ./cmd/lambda
	cd $(DIST_DIR)/authorizer && zip -q -9 ../personal-tools-authorizer.zip bootstrap

build-mcp:
	mkdir -p $(DIST_DIR)/mcp
	cd $(MCP_DIR) && GOCACHE=/private/tmp/personal-tools-mcp-go-cache GOMODCACHE=$(GO_MOD_CACHE) CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ../../$(DIST_DIR)/mcp/bootstrap ./cmd/lambda
	cd $(DIST_DIR)/mcp && zip -q -9 ../personal-tools-mcp.zip bootstrap

build-credentials:
	mkdir -p $(DIST_DIR)/credentials
	cd $(CREDENTIALS_DIR) && GOCACHE=/private/tmp/personal-tools-credentials-go-cache GOMODCACHE=$(GO_MOD_CACHE) CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ../../$(DIST_DIR)/credentials/bootstrap ./cmd/lambda
	cd $(DIST_DIR)/credentials && zip -q -9 ../personal-tools-credentials.zip bootstrap

build-credential-agent:
	mkdir -p $(DIST_DIR)/credential-agent
	cd $(CREDENTIAL_AGENT_DIR) && GOCACHE=/private/tmp/personal-tools-credential-agent-go-cache GOMODCACHE=$(GO_MOD_CACHE) CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ../../$(DIST_DIR)/credential-agent/hermes-credential-provisioner ./cmd/hermes-credential-provisioner
	cd $(CREDENTIAL_AGENT_DIR) && GOCACHE=/private/tmp/personal-tools-credential-agent-go-cache GOMODCACHE=$(GO_MOD_CACHE) CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ../../$(DIST_DIR)/credential-agent/git-credential-hermes ./cmd/git-credential-hermes
	cd $(DIST_DIR)/credential-agent && zip -q -9 ../personal-tools-credential-agent.zip hermes-credential-provisioner git-credential-hermes

clean:
	rm -f \
		$(DIST_DIR)/authorizer/bootstrap \
		$(DIST_DIR)/mcp/bootstrap \
		$(DIST_DIR)/credentials/bootstrap \
		$(DIST_DIR)/credential-agent/hermes-credential-provisioner \
		$(DIST_DIR)/credential-agent/git-credential-hermes \
		$(DIST_DIR)/personal-tools-authorizer.zip \
		$(DIST_DIR)/personal-tools-mcp.zip \
		$(DIST_DIR)/personal-tools-credentials.zip \
		$(DIST_DIR)/personal-tools-credential-agent.zip
