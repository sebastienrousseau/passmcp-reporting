# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0

.PHONY: all build test test-race coverage coverage-badge vet lint format spdx-check spec spec-verify readme-check \
        example-check integrations lockstep family api-check apidoc help name-guard release-versions demo

# Every gate CI runs, in the order the cheap ones fail first.
all: format vet lint spdx-check spec-verify example-check test integrations

build:
	go build ./...

test:
	go test ./... -cover

test-race:
	go test -race -shuffle=on ./...

# The gate is 85% statement coverage in every package with statements.
coverage:
	go test -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

# The shields.io endpoint the README's coverage badge reads, measured the
# way the Coverage Gate measures (example programs excluded). The Manual
# workflow publishes it to GitHub Pages from main.
BADGE ?= coverage.json
coverage-badge:
	go test -count=1 -coverprofile=coverage.out ./...
	go run ./scripts/coveragebadge -profile coverage.out -exclude /examples/ > $(BADGE)

vet:
	go vet ./...

lint:
	golangci-lint run ./...

format:
	gofmt -l -w .

# The README follows the portfolio template: headings in order, no
# unresolved {{VARIABLES}} (AGENTS.md §7.3).
readme-check:
	scripts/readme-check.sh

spdx-check:
	go run ./scripts/spdx_sweep.go

# The published schemas are generated from the attestation, a2a and graph
# types.
spec:
	go run ./scripts/specgen/main.go

spec-verify:
	go run ./scripts/specgen/main.go -check

example-check:
	go vet ./examples/... && go run ./examples/verify attestation/testdata/statement.json >/dev/null && go run ./examples/verify a2a/testdata/statement.json >/dev/null

# The nested modules under integrations/ are programs built from this
# checkout; ./... from the root does not see them, so they get their own
# gate with the same linter configuration.
# Every exported identifier in the public packages has a doc comment, since
# pkgsite renders those comments as the API reference.
# The README demo (.github/demo.gif), rendered by VHS from .github/demo.tape:
# examples/verify checking both sample attestations offline. The build warms
# the cache so the recorded `go run` starts at once. Needs vhs, ttyd and
# ffmpeg.
demo:
	go build -o build/demo/verify ./examples/verify
	vhs .github/demo.tape

apidoc:
	go run ./scripts/apidoc ./attestation ./a2a ./graph ./spec

integrations:
	cd integrations/agentgateway-extmcp && go vet ./... && go test ./... -cover && golangci-lint run --config ../../.golangci.yml ./...
	# Installable as published: no replace, and a module-mode build succeeds.
	! grep -q '^replace' integrations/agentgateway-extmcp/go.mod
	cd integrations/agentgateway-extmcp && GOWORK=off go build ./...
	# Shell completions generate from the flag set and parse.
	$(MAKE) -C integrations/agentgateway-extmcp completions

# Every version-bearing place agrees with the newest CHANGELOG heading.
release-versions:
	scripts/verify-release-versions.sh "v$$(grep -Eo '^## \[[0-9]+\.[0-9]+\.[0-9]+\]' CHANGELOG.md | head -1 | tr -d '#[] ')"

# This repository carries passmcp's version. See docs/ecosystem.md in passmcp.
lockstep:
	scripts/lockstep.sh

# The family manifest in passmcp is the single source of what this repository is.
family:
	scripts/family.sh

# The root module's tags are bare `vX.Y.Z`; the nested processor's are
# `integrations/agentgateway-extmcp/vX.Y.Z`. Without --match, describe picked
# whichever was newest - the nested tag, which gorelease cannot use as this
# module's base - and the check reported "no incompatible change" having
# compared nothing. A gorelease that did not run is now a failure, not a pass.
api-check:
	@tag=$$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || true); \
	if [ -z "$$tag" ]; then echo "api-check: no release tag yet, nothing to compare"; exit 0; fi; \
	out=$$(go run golang.org/x/exp/cmd/gorelease@latest -base="$$tag" 2>&1); rc=$$?; \
	printf '%s\n' "$$out"; \
	if printf '%s' "$$out" | grep -qiE 'incompatible changes'; then \
	  echo "api-check: the public API changed incompatibly against $$tag"; exit 1; \
	fi; \
	if [ "$$rc" -ne 0 ]; then echo "api-check: gorelease did not complete against $$tag"; exit 1; fi; \
	echo "api-check: no incompatible change against $$tag"

help:
	@printf '%s\n' "targets: all build test test-race coverage coverage-badge vet lint format spdx-check spec spec-verify example-check integrations lockstep family api-check apidoc readme-check name-guard release-versions demo"

# The project was renamed to passmcp: the old name may appear only in the
# provenance line (scripts/name-guard.sh).
name-guard:
	./scripts/name-guard.sh
