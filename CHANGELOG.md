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

## [0.0.1] — Unreleased

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
