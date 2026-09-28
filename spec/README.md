<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The published schemas

[`attestation/mcp-evaluation-v1.schema.json`](attestation/mcp-evaluation-v1.schema.json)
is JSON Schema 2020-12 for an in-toto Statement carrying the
`https://satellion.com/attestation/mcp-evaluation/v1` predicate.

It is generated from the Go types in [`../attestation`](../attestation)
by `make spec`, and CI fails when it drifts. Do not edit it by hand. The
`spec` Go package embeds the same bytes, so a Go program validating by
schema and one importing the verifier agree.

The scoring rubric — weights, deductions, grade bands — is not here. It is
generated from passmcp's scorer and published in passmcp's
[`spec/rubric/`](https://github.com/sebastienrousseau/passmcp/tree/main/spec/rubric),
because the scorer is the one thing it must never disagree with
([ADR 0001](../docs/adr/0001-verifier-in-its-own-module.md)).

Two more schemas are generated the same way:

- [`attestation/a2a-evaluation-v1.schema.json`](attestation/a2a-evaluation-v1.schema.json),
  for a Statement carrying the `https://satellion.com/attestation/a2a-evaluation/v1`
  predicate, from [`../a2a`](../a2a);
- [`graph/graph-v1.schema.json`](graph/graph-v1.schema.json), for a
  security graph document (`https://satellion.com/graph/v1`), from
  [`../graph`](../graph).

[docs/format.md](../docs/format.md) describes every field of the MCP
evaluation, [docs/a2a.md](../docs/a2a.md) the A2A one, and
[docs/graph.md](../docs/graph.md) the graph.
