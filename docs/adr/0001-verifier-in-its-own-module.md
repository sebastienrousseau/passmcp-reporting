<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0001 — The verifier is its own module; the rubric and the renderers stay in passmcp

**Status:** Accepted · **Date:** 2026-09-23

## Context

passmcp's [ADR 0011](https://github.com/sebastienrousseau/passmcp/blob/main/docs/adr/0011-attestation-format-is-apache.md)
relicensed the attestation format under Apache-2.0 and extracted the
verifier into `satellion.com/passmcp/attestation`, a package
importing only the standard library. That was meant to let a gateway embed
it. Preparing the first such contribution showed it did not, quite: a Go
consumer that imports a package from passmcp's module records every one of
passmcp's dependencies in its `go.sum` — the terminal UI, the CLI framework —
because Go resolves modules, not packages. A gateway adding a verifier
inherited a dependency graph it never used and its reviewers had to vet.

The family manifest reserves `passmcp-reporting` for "the report schema, the
renderers, the attestation predicate and its offline verifier, plus the
rubric as versioned data".

## Decision

**The `attestation` package moves here unchanged, as the whole of a module
with no `require` directive.** passmcp imports it back and its own
`attestation` package becomes a deprecated forwarder, so no importer
breaks. The predicate's JSON Schema moves with the types it is generated
from, and a `spec` package embeds it.

**The rubric stays in passmcp, for now.** It is generated from the scorer,
and the scorer is the one thing it must never disagree with; a copy here
would drift the day the scorer changed. It moves when passmcp's scorer reads
the rubric as data, at which point the data is the source and belongs
here.

**The renderers stay in passmcp, for now.** No consumer outside passmcp renders
a report today. Extracting them without one would be designing an API for
a user who does not exist, which is how APIs get designed twice.

## Consequences

- passmcp gains its first first-party dependency, and this repository tags
  before passmcp for every release, which the lockstep check allows by
  exactly one patch.
- A consumer's `go.sum` gains one line for a verifier.
- The manifest's description of this repository is ahead of its contents
  by the rubric and the renderers, and says so.

## What would make this wrong

A second producer of statements. If something other than passmcp writes
them, the rubric has to live where both producers read it, and that is
here.
