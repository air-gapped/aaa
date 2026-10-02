# aaa Makefile — `make ci` mirrors .github/workflows/ci.yml; run it before pushing.

SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables --no-builtin-rules
.DEFAULT_GOAL := help

VERSION ?= $(shell git describe --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
BIN_DIR ?= $(HOME)/.local/bin
SKILL_DIR ?= $(HOME)/.claude/skills/aaa

.PHONY: help ci fmt vet test vuln fuzz mutants build install demo clean

##@ 1 · Check (same steps as CI)
ci: fmt vet test vuln ## Run every CI check locally
	@echo "✓ all CI checks passed"

fmt: ## Fail if any file is not gofmt-formatted
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet: ## go vet
	go vet ./...

test: ## Run the tests
	go test ./...

vuln: ## Check dependencies and code for known vulnerabilities
	govulncheck ./...

FUZZTIME ?= 30s
# gremlins v0.6.0 fails on Go 1.25+ (no "covdata" tool, gremlins issue 285); main has the fix.
GREMLINS ?= github.com/go-gremlins/gremlins/cmd/gremlins@b48a4aad1

fuzz: ## Fuzz the argument parser and the terminal sanitizer (FUZZTIME each)
	go test -run '^$$' -fuzz '^FuzzParse$$' -fuzztime $(FUZZTIME) .
	go test -run '^$$' -fuzz '^FuzzClean$$' -fuzztime $(FUZZTIME) .

mutants: ## Mutation testing; survivors are untested behaviour
	go run $(GREMLINS) unleash --timeout-coefficient 3 .

##@ 2 · Build and install
build: ## Build the static binary ./aaa
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o aaa .

install: build ## Install the binary and the agent skill for this user
	install -D -m 755 aaa $(BIN_DIR)/aaa
	install -d $(SKILL_DIR)
	$(BIN_DIR)/aaa --skill > $(SKILL_DIR)/SKILL.md

##@ 3 · Docs
demo: build ## Re-record docs/demo.gif from docs/demo.tape (needs vhs and ttyd)
	PATH="$(CURDIR):$$PATH" vhs docs/demo.tape

##@ Help
clean: ## Remove the built binary
	rm -f aaa

help: ## Show this help
	@awk 'BEGIN{FS=":.*?## "} \
	     /^##@/{printf "\n\033[1m%s\033[0m\n", substr($$0,5); next} \
	     /^[a-zA-Z_0-9-]+:.*?## /{printf "  \033[36m%-10s\033[0m %s\n",$$1,$$2}' $(MAKEFILE_LIST)
