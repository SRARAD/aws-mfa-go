# aws-mfa Makefile
#
# Run `make` or `make help` to see everything.

.DEFAULT_GOAL := help

.PHONY: help fmt vet lint test test-race test-short coverage coverage-badge check \
	build install tidy deps clean install-hooks tools version release

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# Release bump: patch (default), minor, or major. Or set TAG=v0.2.0 explicitly.
BUMP ?= patch

# Optional: `make test PKG=./internal/app/...`
PKG ?= ./...

##@ Getting oriented

help: ## Show this help
	@echo
	@echo "Usage:  make <target>"
	@echo
	@echo "Getting oriented"
	@echo "  help                   Show this help"
	@echo
	@echo "Daily loop (format -> lint -> test)"
	@echo "  fmt                    Format imports/code (goimports-reviser)"
	@echo "  vet                    Static analysis (go vet)"
	@echo "  lint                   Full lint suite (golangci-lint)"
	@echo "  test                   Unit tests"
	@echo "  test-short             Unit tests with -short"
	@echo "  test-race              Unit tests with the race detector"
	@echo "  coverage               Coverage report (writes coverage.out)"
	@echo "  coverage-badge         Write badges/coverage.svg from coverage.out"
	@echo "  check                  Autofix, lint, and unit tests"
	@echo
	@echo "Build & install"
	@echo "  build                  Compile aws-mfa into ./aws-mfa"
	@echo "  install                Install aws-mfa into GOPATH/bin"
	@echo
	@echo "Modules & cleanup"
	@echo "  tidy                   Sync go.mod / go.sum with imports"
	@echo "  deps                   Download module deps"
	@echo "  clean                  Remove binaries and coverage artifacts"
	@echo
	@echo "Release"
	@echo "  version                Show current VERSION file + latest git tag"
	@echo "  release                Bump tag + latest, update VERSION, push (BUMP=patch|minor|major)"
	@echo
	@echo "Tooling"
	@echo "  install-hooks          Install git pre-commit (autofix + lint + test)"
	@echo "  tools                  Install goimports-reviser + golangci-lint v2"
	@echo

##@ Daily loop (format → lint → test)

fmt: ## Autofix imports/code (goimports-reviser + golangci-lint fmt/fix)
	goimports-reviser -format -recursive .
	-golangci-lint fmt ./...
	-golangci-lint run --fix ./...

vet: ## Static analysis (go vet)
	go vet ./...

lint: ## Full lint suite (golangci-lint; no write)
	golangci-lint run ./...

test: ## Unit tests (PKG=./path/... for one package)
	go test $(PKG)

test-short: ## Unit tests with -short
	go test -short $(PKG)

test-race: ## Unit tests with the race detector
	go test -race $(PKG)

# Default coverage scope excludes CLI mains (cmd/, internal/cli) that drag totals down.
# Override: make coverage COVERAGE_PKG=./...
COVERAGE_PKG ?= ./internal/app ./internal/awsapi ./internal/creds ./internal/prompt

coverage: ## Tests + coverage report (writes coverage.out)
	go test -cover "-coverprofile=coverage.out" $(COVERAGE_PKG)
	go tool cover "-func=coverage.out"

coverage-badge: coverage ## Write shields-style SVG from coverage.out
	./scripts/coverage-badge.sh coverage.out badges/coverage.svg

check: fmt lint test ## Autofix, lint, test (matches pre-commit)

##@ Build & install

build: ## Compile aws-mfa into ./aws-mfa
	go build -ldflags "$(LDFLAGS)" -o aws-mfa ./cmd/aws-mfa

install: ## Install aws-mfa into $$GOPATH/bin (or $$GOBIN)
	go install -ldflags "$(LDFLAGS)" ./cmd/aws-mfa

##@ Modules & cleanup

tidy: ## Sync go.mod / go.sum with imports
	go mod tidy

deps: ## Download module deps into the module cache
	go mod download

clean: ## Remove built binaries and coverage artifacts
	go clean ./...
	rm -rf bin badges
	rm -f aws-mfa coverage.out coverage.txt

##@ Release

version: ## Show VERSION file and latest git tag / next patch
	@go run ./cmd/release -dry-run

# Bump semver, commit VERSION, annotated-tag (v* + floating latest), push (triggers GoReleaser).
# Examples:
#   make release
#   make release BUMP=minor
#   make release BUMP=major
#   make release TAG=v0.2.0
#   make release DRY_RUN=1
release: ## Bump version + latest tags, update VERSION, push (BUMP=patch|minor|major)
	go run ./cmd/release \
		$(if $(TAG),-version=$(TAG),-bump=$(BUMP)) \
		$(if $(DRY_RUN),-dry-run,) \
		$(if $(SKIP_PUSH),-skip-push,) \
		$(if $(ALLOW_DIRTY),-allow-dirty,)

##@ Tooling

install-hooks: ## Install git pre-commit hook (autofix + lint + test)
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Installed .git/hooks/pre-commit"

tools: ## Install goimports-reviser + golangci-lint v2 into $$GOBIN
	go install github.com/incu6us/goimports-reviser/v3@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
	@echo "Installed tools. Ensure GOPATH/bin is on PATH, then: golangci-lint version"
