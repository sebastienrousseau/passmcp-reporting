<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# The statement

A passmcp attestation is an [in-toto v1 Statement](https://github.com/in-toto/attestation)
whose predicate type is `https://satellion.com/attestation/mcp-evaluation/v1`.
It is a claim, not a document: a document is read by a person, a claim is
verified by a machine that was not present when the run happened — a
gateway deciding whether to route to a server, a registry deciding what to
display, an auditor deciding whether a control was met in March.

Four properties are deliberate, and each is a constraint rather than a
nice-to-have.

- **It states what it was judged against.** A verdict with no basis has no
  shelf life: `82/100` means nothing in 2029 unless the rubric and the
  specification revision are attached to it.
- **The subject digest is over a target descriptor, and the statement says
  so.** A digest that looked like an artifact hash while covering a URL
  would be the kind of lie that survives review.
- **It carries the verdict for every check that ran, not only the
  failures.** A consumer cannot otherwise tell "checked and fine" from
  "not checked", which for a conformance tool is the whole difference.
- **It verifies offline.** A gateway must never have to call passmcp to trust
  a statement passmcp produced.

## The envelope

```json
{
  "_type": "https://in-toto.io/Statement/v1",
  "subject": [{ "name": "https://mcp.example.com/mcp", "digest": { "sha256": "…" } }],
  "predicateType": "https://satellion.com/attestation/mcp-evaluation/v1",
  "predicate": { … }
}
```

Exactly one subject. Its `name` is the target's endpoint or command line,
and its digest is over the canonical descriptor `transport + "\n" +
endpoint` — not over an artifact, which `predicate.subjectKind`
(`mcp-target-descriptor`) says out loud. A verifier recomputes the digest
from the predicate's target and refuses a statement whose subject was
renamed.

The digest is an in-toto `DigestSet`, so it may carry more than one
algorithm. The verifier knows `sha256` and `sha512`: it recomputes every
one of those the subject carries and refuses the statement if any does not
match, and it needs at least one of them. A subject with only an algorithm
it does not know, `md5` say, is refused as a subject with no digest is;
an unknown algorithm beside a known one is carried and ignored. passmcp
writes `sha256` alone
([ADR 0003](adr/0003-subject-digest-sha256-or-sha512.md)).

## The predicate

| Field | What it is | Why it is there |
|---|---|---|
| `subjectKind` | always `mcp-target-descriptor` | so nobody reads the digest as an artifact hash |
| `target` | `transport` (`http` or `stdio`), `endpoint`, and what the server said it was | what the claim is about |
| `judgedAgainst` | `specRevision`, `rubric`, `checkInventory` | a score is only comparable under the same rubric; a verdict only meaningful against the revision it was judged by |
| `instrument` | the tool and version that produced it | what to reproduce it with |
| `ranAt`, `took` | when, and for how long | a verdict about a live service is a verdict about a moment |
| `verdicts` | every check that ran: `id`, `phase`, `status`, `severity`, `evidence`, `doc` | the whole picture, passes included, with the request references that produced each |
| `counts` | the verdicts by status | the summary a policy engine gates on |
| `score` | `total`, `grade`, categories assessed and total, per-category scores | the rating, carried only with its rubric version |
| `blocked` | why the run stopped early, when it did | a partial run cannot pass for a full one |
| `plan` | the run specification with secret values removed, the credential mode, what was given by value, and OS, architecture and kernel | what to repeat: passmcp's `--reproduce` runs it and reports drift |

`verdict.status` is one of `pass`, `warn`, `fail`, `skip`, `info`. A `pass`
is issued only on the evidence of a request that showed the property; an
`info` records an observation without judgement; a `skip` names its
reason. That discipline is passmcp's, recorded in its
[ADR 0002](https://github.com/sebastienrousseau/passmcp/blob/main/docs/adr/0002-findings-cite-requests.md),
and it is why the `evidence` references are worth carrying.

## The rubric

The score is computed under a rubric that passmcp publishes as data at
[`spec/rubric/rubric-v1.json`](https://github.com/sebastienrousseau/passmcp/blob/main/spec/rubric/rubric-v1.json),
generated from the scorer itself. Each category starts at 100; each failing
or warning verdict in its phases deducts what the rubric gives for that
outcome; a category counts only if one of its phases ran; the total is the
weight-averaged score of those that did. Anyone can recompute a score from
the verdicts and get the same number, which is what makes it a claim.

## The schema

[`spec/attestation/mcp-evaluation-v1.schema.json`](https://github.com/sebastienrousseau/passmcp-reporting/blob/main/spec/attestation/mcp-evaluation-v1.schema.json)
is JSON Schema 2020-12 for the whole statement, generated from the Go
types by reflection, with the constants and enumerations fixed. A consumer
with its own schema validator uses it directly; the `spec` package embeds
the same bytes for a Go program.
