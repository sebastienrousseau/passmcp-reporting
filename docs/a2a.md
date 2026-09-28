<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The A2A evaluation

An A2A evaluation is passmcp's claim about one agent that speaks the
Agent2Agent protocol, in the same in-toto envelope as an MCP evaluation.
Its predicate type is `https://satellion.com/attestation/a2a-evaluation/v1`,
and the `a2a` package reads and verifies it offline.

It is a separate predicate type, not an MCP evaluation with a protocol
field, so a consumer that knows only MCP evaluations refuses it rather than
reading its verdicts as a server's.

## What it carries

The same shape as an [MCP evaluation](format.md), with three differences:

- **The subject** is the agent's base URL, digested as the descriptor
  `a2a\n<endpoint>` and marked `subjectKind: a2a-target-descriptor`. The
  prefix keeps an agent and an MCP server at the same URL from sharing a
  digest.
- **The target** has a transport of `https` or `http` that must match the
  endpoint's scheme, and `agent` records what the Agent Card said the agent
  was.
- **`card`** describes the Agent Card the verdicts came from: where it was
  fetched, the SHA-256 of its JCS canonical form, whether it was signed,
  and the key that verified the signature.

`judgedAgainst.protocolVersion` is the A2A version the card declared, in
place of the MCP revision.

## Verifying one

```go
st, err := a2a.Parse(b) // refuses anything that is not an A2A evaluation
if err != nil {
    return deny(err)
}
if v, ok := st.VerdictFor("a2a.card_signature"); !ok || v.Status != "pass" {
    return deny("the agent's card is not verified")
}
```

`Validate` applies the MCP evaluation's rules: one subject whose name and
digest match the target, an identified instrument and run time, a check
inventory, a rubric wherever there is a score, and counts that are the
verdicts' own. The schema is
[`spec/attestation/a2a-evaluation-v1.schema.json`](https://github.com/sebastienrousseau/passmcp-reporting/blob/main/spec/attestation/a2a-evaluation-v1.schema.json).
