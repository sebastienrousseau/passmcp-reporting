<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0002 — The agentgateway processor is a nested module, built from the tree through `go.work`

**Status:** Accepted · **Date:** 2026-09-28

This record writes down a decision already in the tree: the comment in
`go.work`, the `integrations` target in the `Makefile` and the
processor's README each state part of it.

## Context

The agentgateway processor needs gRPC and protobuf. The verifier's whole
reason to be a separate module ([ADR 0001](0001-verifier-in-its-own-module.md))
is that importing it adds nothing to a consumer's `go.sum`. Putting the
processor in the root module would put gRPC in every verifier consumer's
dependency graph.

Built from this checkout, the processor must compile against the verifier
at the same commit, or a change to both could not be tested together. The
usual way to do that, a `replace` directive in the processor's `go.mod`,
makes the module uninstallable: `go install` refuses any module whose
`go.mod` carries one.

## Decision

**The processor is its own module under `integrations/agentgateway-extmcp`,
tagged `integrations/agentgateway-extmcp/vX.Y.Z`.** Its `go.mod` requires
a released version of `satellion.com/passmcp-reporting`, never a
pseudo-version or a `replace`.

**The root `go.work` joins the two modules inside a checkout.** A workspace
file is never part of a published module, so the tree builds against the
same commit while the published module builds against the tagged release.

## Consequences

- `make integrations` and CI fail when the processor's `go.mod` carries a
  `replace`, and CI builds it with `GOWORK=off` the way `go install` would.
- `scripts/verify-release-versions.sh` fails when the processor requires
  a pseudo-version, or a root release newer than the one being cut. It
  cannot require the release being cut: that tag does not exist until
  the commit carrying the requirement is tagged.
- A verifier change the processor depends on ships in two steps: the root
  module is tagged first, then the processor's `go.mod` is moved to that
  release.

## What would make this wrong

A Go toolchain that lets `go install` honour a `replace` directive, or a
processor that no longer needs anything outside the standard library; in
either case the second module costs more than it saves.
