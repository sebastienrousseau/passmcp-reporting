<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The security graph

The `graph` package is the data model of passmcp's security graph: which
agents reach which MCP servers and tools, under which identities, where each
server was discovered, and what its attestation said. passmcp-graph stores and
queries it; passmcp's discovery writes the endpoints it finds into it. The
schema is [`spec/graph/graph-v1.schema.json`](https://github.com/sebastienrousseau/passmcp-reporting/blob/main/spec/graph/graph-v1.schema.json).

## Nodes and edges

| Node kind | Properties |
| :--- | :--- |
| `agent` | the client product and the configuration file it came from |
| `server` | transport, endpoint, name, version, how it admits clients, and from the latest attestation its score, grade, digest and failing checks |
| `tool` | name, `readOnlyHint`, `destructiveHint` (absent means undeclared) |
| `identity` | issuer and public client ID |
| `source` | where servers were discovered: a configuration, a gateway, a registry namespace or a target list |
| `attestation` | the statement's digest, predicate type, run time and file |

| Edge | From | To | Carries |
| :--- | :--- | :--- | :--- |
| `uses` | agent | server | the configuration file and key it was declared under |
| `exposes` | server | tool | — |
| `authorizes` | identity | server | the scopes granted |
| `discovered_by` | server | source | — |
| `attested_by` | server | attestation | — |

## Three properties

**IDs are derived, not assigned.** A server's ID is `server:` followed by
the same digest its attestation subject carries; a tool's is derived from
its server and name; an agent's from its client, configuration file and
name. Ingesting the same evidence twice therefore yields the same nodes, and
`Upsert` and `Link` merge rather than duplicate, so the encoding is
byte-identical.

**No field can hold a secret.** A client configuration's tokens, API keys
and environment values have nowhere to go: a `uses` edge keeps only the file
and the key. There is no free-form property bag either. A test walks every
type in the model to prove it.

**The encoding is canonical.** Nodes are sorted by ID and edges by source,
kind and target, so a store under version control diffs cleanly.

## The store

A store is a directory holding one file, `graph.json`. `Load` reads it (an
absent store is an empty graph) and `Save` validates the graph and replaces
the file atomically. Tools may keep their own files beside it.
