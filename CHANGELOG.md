<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Changelog

All notable changes to passmcp-reporting are documented here. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
versions are [Semantic Versioning](https://semver.org/) shaped.

**This repository carries passmcp's version.** It is in lockstep with
[passmcp](https://github.com/sebastienrousseau/passmcp): every release here is
a passmcp release, whatever changed in this tree, and a release here with
nothing in it is the version rule working. Because passmcp imports this
module, this repository tags first.

## [0.0.2] — 2026-09-29

The family's second release. Nothing a consumer imports changed: the
`attestation`, `a2a`, `graph` and `spec` packages, the schemas and the
predicates are the 0.0.1 ones, and `Validate` accepts exactly what it
accepted.

### Changed

- **The agentgateway processor requires the released verifier.** Its
  `go.mod` names `satellion.com/passmcp-reporting v0.0.1` instead of a
  pseudo-version of an untagged commit, and
  `scripts/verify-release-versions.sh` fails when it names a
  pseudo-version or a release newer than the one being cut
  ([ADR 0002](docs/adr/0002-processor-nested-module-via-go-work.md)).
- **The processor's gRPC moves to v1.83.2, clearing GO-2026-6443**
  (CVE-2026-84445): a client that omits the `:authority` and `Host`
  headers could panic the processor's gRPC server. v1.84.0 has no fixed
  release; v1.83.2 is the fixed release on the line before it. CI now
  runs govulncheck on the processor module too, which `./...` from the
  root never reached.
- **`make family` accepts the family standard's status names.** The
  manifest's `released` counts as `shipping` did, and both are accepted
  while passmcp's main still serves the earlier manifest.
- **Every version-bearing place is checked on every pull request.** CI
  runs `make release-versions`, which now also covers the release notes,
  `CITATION.cff` and the README's family sentence.

### Added

- **A coverage badge measured by CI.** `make coverage-badge` writes the
  shields.io endpoint document from the cover profile, and the Manual
  workflow publishes it with the manual on every push to main.
- **An OpenSSF Scorecard workflow and a devcontainer** that boots to a
  working `make build test` with the linter CI pins.
- **The architecture overview in the manual**, moved to
  `docs/ARCHITECTURE.md`, and ADR 0002 recording why the processor is a
  nested module.

## [0.0.1] — 2026-09-29

The first release.

### Added

- **The MCP evaluation attestation and its offline verifier.** The in-toto
  statement a passmcp run writes about an MCP server, its JSON Schema, and
  the verifier a gateway, registry or CI system embeds to check one
  offline, under Apache-2.0.
- **The security graph's data model, `graph`.** Agents, MCP servers, tools,
  identities, discovery sources and attestations, joined by `uses`,
  `exposes`, `authorizes`, `discovered_by` and `attested_by` edges, for
  passmcp-graph to store and passmcp's discovery to write into. IDs are derived
  from what a node is: a server's ID carries the digest its attestation
  subject covers, so ingesting the same evidence twice changes nothing,
  and `Upsert` and `Link` merge rather than duplicate. No type has a field
  a secret could be kept in, and a test walks the model to prove it. The
  encoding is canonical, and a store is one `graph.json` in a directory,
  replaced atomically. The schema is `spec/graph/graph-v1.schema.json`,
  embedded as `spec.GraphSchema`.
- **The A2A evaluation predicate, `a2a`.** The same in-toto envelope and
  verdicts about an agent that speaks the Agent2Agent protocol, under its
  own predicate type (`https://satellion.com/attestation/a2a-evaluation/v1`)
  so an MCP-only consumer refuses it rather than misreading it. `Parse` and
  `Validate` apply the MCP evaluation's rules; the statement records the
  Agent Card's canonical digest, whether it was signed and the key that
  verified it. The schema is `spec/attestation/a2a-evaluation-v1.schema.json`,
  embedded as `spec.A2AAttestationSchema`, and `examples/verify` reads
  either predicate.
