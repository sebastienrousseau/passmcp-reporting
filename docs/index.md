<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# passmcp-reporting documentation

The attestation a passmcp run makes about one MCP server, and the verifier
that checks it offline.

| Document | Covers |
|---|---|
| [format.md](format.md) | Every field of a statement, and why it is there |
| [verify.md](verify.md) | Verifying a statement: in Go, by schema, and the signature around it |
| [BENCHMARKS.md](BENCHMARKS.md) | What an admission decision costs |
| [adr/](adr/README.md) | Decision records for this repository |
| [../spec/](https://github.com/sebastienrousseau/passmcp-reporting/blob/main/spec/README.md) | The published schema, and where the rubric is |

Producing a statement is passmcp's job; its manual is at
<https://satellion.com/passmcp/docs/>. The API reference is at
<https://pkg.go.dev/satellion.com/passmcp-reporting>.
