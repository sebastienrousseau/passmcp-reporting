# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0

.PHONY: all build test test-race vet lint generate generate-check completions image help

# The gates, cheapest first. This module has its own go.mod; run make here,
# not from the repository root.
all: vet lint test

build:
	go build ./...

test:
	go test ./... -cover

test-race:
	go test -race -shuffle=on -count=1 ./...

vet:
	go vet ./...

# The lint configuration is the repository's; a nested module is held to
# the same rules as the verifier it wraps.
lint:
	golangci-lint run --config ../../.golangci.yml ./...

# Regenerate gen/extmcp from proto/ext_mcp.proto. buf compiles, the two
# protoc plugins are fetched at pinned versions into a scratch GOBIN, and
# the SPDX header the repository's spdx-check wants is prepended, since
# protoc-gen-go has no option for one.
BUF        := go run github.com/bufbuild/buf/cmd/buf@v1.66.0
GEN_GOBIN  := $(CURDIR)/build/bin
generate:
	mkdir -p $(GEN_GOBIN)
	GOBIN=$(GEN_GOBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
	GOBIN=$(GEN_GOBIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	PATH="$(GEN_GOBIN):$$PATH" $(BUF) generate
	# REUSE-IgnoreStart: the lines below write a header, they are not one.
	for f in gen/extmcp/*.pb.go; do \
	  { printf '%s\n' \
	      '// SPDX-FileCopyrightText: The agentgateway Authors <https://github.com/agentgateway/agentgateway>' \
	      '// SPDX-License-Identifier: Apache-2.0' \
	      '//' \
	      '// Generated from proto/ext_mcp.proto (agentgateway commit 38fc45b); see proto/README.md.' \
	      ''; cat "$$f"; } > "$$f.tmp" && mv "$$f.tmp" "$$f"; \
	done
	# REUSE-IgnoreEnd
	gofmt -l gen/

# Fail if the committed bindings differ from what generate produces.
generate-check: generate
	git diff --exit-code -- gen/

# Shell completions, generated from the flag set by the binary itself, into
# build/completions. bash is syntax-checked here; zsh and fish when present.
completions:
	mkdir -p build/completions
	go run ./cmd/agentgateway-extmcp -completion bash > build/completions/agentgateway-extmcp.bash
	go run ./cmd/agentgateway-extmcp -completion zsh > build/completions/_agentgateway-extmcp
	go run ./cmd/agentgateway-extmcp -completion fish > build/completions/agentgateway-extmcp.fish
	bash -n build/completions/agentgateway-extmcp.bash
	if command -v zsh >/dev/null; then zsh -n build/completions/_agentgateway-extmcp; fi
	if command -v fish >/dev/null; then fish -n build/completions/agentgateway-extmcp.fish; fi

# The container image, built from this directory alone as go install sees it.
image:
	docker build --tag agentgateway-extmcp:dev .

help:
	@grep -E '^[a-z-]+:' $(MAKEFILE_LIST) | cut -d: -f1 | sort -u
