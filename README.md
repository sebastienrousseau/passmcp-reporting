<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<p align="center">
  <img src="https://raw.githubusercontent.com/sebastienrousseau/passmcp/main/.github/logo.svg" alt="passmcp-reporting logo" width="128" />
</p>

<h1 align="center">passmcp-reporting</h1>

<p align="center">
  The attestation a passmcp run makes about a Model Context Protocol server, and the verifier that checks one offline — licensed so a gateway, registry or CI system can embed it.
</p>

<p align="center">
  <a href="https://github.com/sebastienrousseau/passmcp-reporting/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/sebastienrousseau/passmcp-reporting/ci.yml?branch=main&style=for-the-badge&logo=github&label=Build" alt="Build" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-reporting/blob/main/DEVELOPMENT.md#coverage"><img src="https://img.shields.io/endpoint?url=https%3A%2F%2Fsebastienrousseau.com%2Fpassmcp-reporting%2Fcoverage.json&style=for-the-badge&logo=codecov&logoColor=white" alt="Coverage" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-reporting/releases"><img src="https://img.shields.io/github/v/release/sebastienrousseau/passmcp-reporting?style=for-the-badge&color=fc8d62&logo=github&label=Release" alt="Release" /></a>
  <a href="https://pkg.go.dev/satellion.com/passmcp-reporting"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?style=for-the-badge&labelColor=555555&logo=go&logoColor=white" alt="Docs" /></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/sebastienrousseau/passmcp-reporting"><img src="https://img.shields.io/ossf-scorecard/github.com/sebastienrousseau/passmcp-reporting?style=for-the-badge&label=OpenSSF%20Scorecard&logo=openssf" alt="OpenSSF Scorecard" /></a>
  <a href="https://www.bestpractices.dev/projects/15109"><img src="https://www.bestpractices.dev/projects/15109/badge" alt="OpenSSF Best Practices" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue.svg?style=for-the-badge" alt="License: Apache-2.0" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-reporting/blob/main/DEVELOPMENT.md#requirements"><img src="https://img.shields.io/badge/go-1.26.8%2B-93450a.svg?style=for-the-badge&logo=go" alt="Go 1.26.8+" /></a>
</p>

<p align="center">
  <img src=".github/demo.gif" alt="examples/verify checking the two sample attestations offline: an MCP evaluation scored 88 out of 100 and an A2A evaluation scored 62, each with its failing check" width="100%" />
</p>

---

## Contents

**Getting started**

- [Install](#install) — `go get`, or the schema alone
- [Requirements](#requirements) — the Go floor, and nothing else
- [Quick Start](#quick-start) — verify a statement in twelve lines

**The passmcp-reporting ecosystem**

- [The passmcp-reporting ecosystem](#the-passmcp-reporting-ecosystem) — `passmcp`, `passmcp-reporting`, `passmcp-server`, `passmcp-action`, `passmcp-graph`, `passmcp-registry`, `passmcp-lsp`, `passmcp-census` and `satellion.com` at a glance

**Library reference**

- [Capabilities at a glance](#capabilities-at-a-glance) — the current surface by theme
- [Ecosystem comparison](#ecosystem-comparison) — what this is beside, and is not
- [Benchmarks](#benchmarks) — what an admission decision costs
- [Features](#features) — what a statement carries and what the verifier checks
- [Configuration](#configuration) — there is none
- [Examples](#examples) — runnable example index

**Operational**

- [When not to use passmcp-reporting](#when-not-to-use-passmcp-reporting) — limitations
- [Development](#development) — make targets, what is generated, CI
- [Security](#security) — what `Validate` guarantees and what it does not
- [Documentation](#documentation) — all reference docs
- [Stability guarantees](#stability-guarantees) — the breaking axis is what `Validate` accepts
- [License](#license)

---

## Install

### As a Go library

```sh
go get satellion.com/passmcp-reporting@v0.0.4
```

The module has no dependencies: adding it adds one line to `go.sum`.

### The schema alone

A consumer that validates by JSON Schema rather than by importing Go takes
[`spec/attestation/mcp-evaluation-v1.schema.json`](spec/attestation/mcp-evaluation-v1.schema.json)
(JSON Schema 2020-12). It is generated from the Go types and CI fails when
the two disagree, so both routes accept the same statements.

---

## Requirements

| Requirement | Floor | Enforced by |
|---|---|---|
| Go | the `go` directive in [`go.mod`](go.mod) | CI tests on that version and on latest stable, on Linux, macOS and Windows |
| Dependencies | none | `go.mod` has no `require` directive, and a pull request that adds one is declined |
| Network | none | the verifier reads bytes it is given and nothing else |

The floor is raised only when a release needs a language feature, on a
patch release like everything else pre-1.0, and the changelog says so.

---

## Quick Start

```go
package main

import (
    "fmt"
    "os"

    "satellion.com/passmcp-reporting/attestation"
)

func main() {
    b, _ := os.ReadFile("statement.json")
    st, err := attestation.Parse(b)
    if err != nil {
        panic(err)
    }
    if err := st.Validate(); err != nil {
        panic(err) // the digest, the subject, the predicate type or a required field is wrong
    }
    if !st.Covers("http", "https://mcp.example.com/mcp") {
        panic("about a different server")
    }
    fmt.Println(st.Predicate.Score.Total, st.Predicate.Score.Grade)
}
```

`Parse` reads the in-toto envelope. `Validate` recomputes the subject digest
from the predicate's target and refuses a statement whose displayed subject
was rewritten, a score without its rubric version, or a predicate of another
type. `Covers` asks whether the statement is about the target you are about
to trust. What you do with the score is your policy; the statement carries
the verdict of every check that ran, so a gate can be as specific as
"no `fail` in `auth.*`".

A statement is produced by [passmcp](https://github.com/sebastienrousseau/passmcp):
`passmcp check … --output attestation`, or `passmcp attest report.json`. Signing
it, and checking the signature, is the envelope's job — see
[docs/verify.md](docs/verify.md).

---

## The passmcp-reporting ecosystem

Every component is released at **0.0.4** and moves in lockstep: one version across the family, released together ([docs/ecosystem.md](https://github.com/sebastienrousseau/passmcp/blob/main/docs/ecosystem.md)).

| Component | Purpose | Use case |
| :--- | :--- | :--- |
| [passmcp](https://github.com/sebastienrousseau/passmcp) | The MCP server diagnostic: checks in nine phases, every finding tied to the request that showed it, signed attestations | Test a server before your agents trust it, and gate it in CI |
| [passmcp-reporting](https://github.com/sebastienrousseau/passmcp-reporting) | The attestation format, its JSON Schemas and offline verifier, the graph model, and the agentgateway processor | Verify an attestation in a gateway, registry or pipeline |
| [passmcp-server](https://github.com/sebastienrousseau/passmcp-server) | passmcp's diagnostics as read-only MCP tools | Evaluate a server, or check an attestation, from inside the agent |
| [passmcp-action](https://github.com/sebastienrousseau/passmcp-action) | passmcp in GitHub Actions and GitLab CI, the image pinned by digest | Fail a build on the findings you choose |
| [passmcp-graph](https://github.com/sebastienrousseau/passmcp-graph) | A local graph of agents, servers, tools and identities built from attestations | Find inherited risk and over-privilege, and gate on policy |
| [passmcp-registry](https://github.com/sebastienrousseau/passmcp-registry) | A signed public scorecard of the MCP Registry's remote servers | Check a public server's standing before connecting to it |
| [passmcp-lsp](https://github.com/sebastienrousseau/passmcp-lsp) | A language server for MCP artefacts, with check-id hover from the guidance catalogue | Catch mistakes in server.json, tool schemas and client configuration while editing |
| [passmcp-census](https://github.com/sebastienrousseau/passmcp-census) | The published reliability census: dataset, methodology, disclosure log and reproduction command | Cite ecosystem-wide reliability figures, and reproduce them |
| [satellion.com](https://github.com/sebastienrousseau/satellion.github.io) | The website, the Go module paths and the format URIs | Read the manual, and resolve `satellion.com/...` imports |

---

## Capabilities at a glance

| Area | Capability | Status |
| :--- | :--- | :--- |
| Read | `Parse` an in-toto v1 Statement carrying the `mcp-evaluation/v1` predicate | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-reporting/releases/tag/v0.0.1) |
| Integrity | `Validate` recomputes the subject digest from the target descriptor and checks every required field | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-reporting/releases/tag/v0.0.1) |
| Scope | `Covers(transport, endpoint)` says whether a statement is about a given target | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-reporting/releases/tag/v0.0.1) |
| Lookup | `VerdictFor(id)` returns one check's outcome, or `ErrNoSuchCheck` | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-reporting/releases/tag/v0.0.1) |
| Drift | `Compare(before, after)` names what got worse, what got better, what appeared and what disappeared | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-reporting/releases/tag/v0.0.1) |
| Produce | `SubjectFor(target)` digests a target exactly as `Validate` recomputes it | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-reporting/releases/tag/v0.0.1) |
| Schema | `spec.AttestationSchema`, the same bytes as the committed file | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-reporting/releases/tag/v0.0.1) |
| A2A | `a2a.Parse` and `Validate` for the `a2a-evaluation/v1` predicate: the same envelope and verdicts about an Agent2Agent agent | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-reporting/releases/tag/v0.0.1) |
| Graph | `graph`: the security graph's model, stable IDs, idempotent `Upsert`/`Link`, canonical encoding and a one-file store; `spec.GraphSchema` | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-reporting/releases/tag/v0.0.1) |
| Gateway | `integrations/agentgateway-extmcp`, an `ExtMcp` processor for agentgateway built on this package, tagged `integrations/agentgateway-extmcp/v0.0.1` | [Released in 0.0.1](https://github.com/sebastienrousseau/passmcp-reporting/releases/tag/integrations%2Fagentgateway-extmcp%2Fv0.0.1) |

Verifying who wrote the bytes is out of scope: that is the job of the
signed envelope around a statement ([docs/verify.md](docs/verify.md)).

---

## Ecosystem comparison

There is no other MCP evaluation attestation to compare against: the
official conformance suite reports pass and fail to a terminal and writes
no portable claim, and the registries display what a server says about
itself. What this format sits beside is the supply-chain machinery it
reuses, and the comparison worth making is with those envelopes.

| Project | Portable claim | Verifies offline | About an MCP server |
| :--- | :---: | :---: | :---: |
| **passmcp-reporting** | yes | yes | yes |
| SLSA provenance | yes | yes | no — about a build |
| CycloneDX / SPDX SBOM | yes | yes | no — about components |
| MCP conformance suite | no | — | yes |

The format is an in-toto Statement so that it verifies with the same tools
and sits in the same policy as the first two rows.

---

## Benchmarks

An admission decision — parse and validate a statement, digest included —
costs microseconds and allocates once per verdict. Measured with
`go test -bench Validate -benchmem` in [`attestation`](attestation/);
numbers and machine in [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md).

| Scenario | Result | Environment |
| :--- | ---: | :--- |
| Parse + Validate, 2-verdict fixture (1.3 KB) | 3.8 µs, 17 allocations | Apple A18 Pro, Go 1.26.8, 2026-09-23 |
| Parse + Validate, 120-verdict statement (28 KB) | 66 µs, 390 allocations | same |

---

## Features

**What a statement carries.** One subject — the evaluated server, digested
as a target descriptor and marked as such — and a predicate with the
target, what it was judged against (MCP revision, rubric version, check
inventory), the instrument, when it ran, the verdict for every check that
ran with its evidence references, the counts, the score, why the run
stopped early if it did, and the plan that produced it with secret values
removed. [docs/format.md](docs/format.md) describes each field and why it
is there.

**What the verifier checks.** The envelope type and predicate type; exactly
one subject, named as the target and digested as the descriptor; every
required field; a rubric version wherever there is a score; a verdict
status from the closed set. It does not check signatures, and it never
follows a reference.

**Two more formats.** The `a2a` package is the same claim about an agent
that speaks the Agent2Agent protocol, under its own predicate type so a
consumer that knows only MCP evaluations refuses it rather than misreading
it. The `graph` package is the security graph passmcp-graph stores and passmcp's
discovery writes: which agents reach which servers and tools, under which
identities, with each server's attested verdict attached. It has no field a
secret could be kept in. [docs/graph.md](docs/graph.md) and
[docs/a2a.md](docs/a2a.md) describe both.

**What the verifier does not do.** Fetch, sign, score or judge. Producing
a statement is passmcp's job; deciding what a score means is yours.

---

## Configuration

None. The verifier is a pure function of the bytes it is given, which is
the property a gateway wants from something it trusts: nothing to
misconfigure, nothing that changes between two machines.

---

## Examples

| Example | Shows |
|---|---|
| [`examples/verify`](examples/verify/main.go) | Read a statement, validate it, print the score and every failing check — the smallest admission hook |
| [`attestation/example_test.go`](attestation/example_test.go) | The package's own runnable example, checked by `go test` |
| [`integrations/agentgateway-extmcp`](integrations/agentgateway-extmcp/README.md) | A complete consumer: an [agentgateway](https://github.com/agentgateway/agentgateway) `mcpGuardrails` processor that denies calls to a backend whose attestation is missing, invalid, below a score floor or failing in a named category. Its own module, so the root stays dependency-free |

```sh
go run ./examples/verify attestation/testdata/statement.json
```

---

## When not to use passmcp-reporting

- **To produce a statement.** Only passmcp writes one; this package reads
  them. `SubjectFor` is exported so a producer digests a target the way
  a verifier recomputes it, not so a second producer exists.
- **As proof of who wrote the bytes.** `Validate` checks integrity and
  structure. A statement that validates could still have been written
  by anyone; sign it and verify the signature with the envelope of your
  choice.
- **To recompute a score.** The rubric is published in passmcp's `spec/`,
  generated from the scorer. This package carries a score and its rubric
  version and does not re-derive it.
- **For anything but MCP evaluations.** The predicate type is fixed; a
  statement of another type is refused rather than guessed at.

---

## Development

```bash
make            # format, vet, lint, headers, schema, example, tests
make test-race  # race detector, randomised order
make spec       # regenerate the schema after a type change
make family     # this repository's row in passmcp's family manifest
make lockstep   # the version is passmcp's, or the next one
make release-versions  # every version-bearing place names the CHANGELOG's version
make coverage-badge    # the coverage.json the badge reads
```

Every gate CI runs has a local form; [DEVELOPMENT.md](DEVELOPMENT.md) maps
them. `spec/attestation/` is generated and never edited by hand.

---

## Security

`Validate` recomputes the subject digest and refuses a rewritten subject, a
second subject, a score without its rubric, or a predicate of another
type; each refusal is a test. The verifier imports only the standard
library and opens no file and no connection; the agentgateway processor,
a network service with its own dependencies, is modelled separately.
CI runs `govulncheck` on both modules on every push. What the verifier
does not check is a signature: that is the envelope around the
statement, and a consumer must verify both. The threat model and the
latest security review are in [`docs/security-model.md`](docs/security-model.md);
verifying a release is [`docs/signing.md`](docs/signing.md).

Report vulnerabilities according to [`SECURITY.md`](SECURITY.md).

---

## Documentation

The four entry points, identical across every repo in the family:

- **[User Manual](https://satellion.com/passmcp/docs/)** — passmcp's rendered manual, including how a statement is produced and signed
- **[API reference](https://pkg.go.dev/satellion.com/passmcp-reporting)** — this module's packages
- **[Developer docs](DEVELOPMENT.md)** — toolchain, task map, reproducing every CI gate locally
- **[Ecosystem map](https://github.com/sebastienrousseau/passmcp/blob/main/docs/ecosystem.md)** — the family, the published artefacts, the lockstep version rule

| Document | Covers |
|---|---|
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | How the format, the verifier, the schema and the processor fit together |
| [`docs/format.md`](docs/format.md) | Every field of a statement, and why it is there |
| [`docs/verify.md`](docs/verify.md) | Verifying one: in Go, by schema, and the signature around it |
| [`docs/a2a.md`](docs/a2a.md) | The A2A evaluation predicate |
| [`docs/graph.md`](docs/graph.md) | The security graph's model and store |
| [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md) | What an admission decision costs |
| [`docs/security-model.md`](docs/security-model.md) | Threat model, trust boundaries, how common weaknesses are countered, and the latest security review |
| [`docs/signing.md`](docs/signing.md) | Verifying a release: the signed tags, the Go checksum, the processor image's provenance |
| [`docs/adr/`](docs/adr/README.md) | Decision records for this repository |
| [`spec/`](spec/README.md) | The published schema, and where the rubric is |
| [`SECURITY.md`](SECURITY.md) | Disclosure policy, supported versions, what is guaranteed |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | Signed-commit and DCO policy, what a format change needs |
| [`CHANGELOG.md`](CHANGELOG.md) | Per-release notes, and the lockstep version rule |
| [`SUPPORT.md`](SUPPORT.md) | Where to ask, and what to expect |

---

## Stability guarantees

passmcp-reporting is pre-1.0, carries passmcp's version, and follows SemVer
with the patch digit moving for everything until 1.0.

**The breaking axis is what `Validate` accepts.** A statement this version
rejects and the previous one accepted — or the reverse — is a breaking
change whether or not a signature moved, because a gateway on either side
of it makes a different admission. Specifically, these are breaking:

- A required field added to, or removed from, the predicate
- A change to how the subject digest is computed
- A change to the predicate type or the statement type
- A verdict status added to or removed from the closed set

Added optional fields, new helpers, and stricter checks that only reject
statements passmcp never wrote are **not** breaking. Every breaking change
lands with a tracking issue open for at least a week and a changelog entry
that names it.

**Deprecation window.** A deprecated function keeps working for at least
one release after the release that announces it.

---

## License

Licensed under the **[Apache License 2.0](LICENSE)**.

The engine that produces these statements, [passmcp](https://github.com/sebastienrousseau/passmcp),
is GPL-3.0-only. This repository is Apache-2.0 so that the format can be
implemented and the verifier embedded without taking on the engine's
licence — passmcp's [ADR 0011](https://github.com/sebastienrousseau/passmcp/blob/main/docs/adr/0011-attestation-format-is-apache.md)
records why.

<p align="right"><a href="#contents">Back to Top</a></p>
