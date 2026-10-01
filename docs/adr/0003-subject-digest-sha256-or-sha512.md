<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0003 — The subject digest may be sha256, sha512 or both

**Status:** Accepted · **Date:** 2026-09-30 · **Tracking issue:**
[#10](https://github.com/sebastienrousseau/passmcp-reporting/issues/10)

## Context

A statement's subject digest is the format's one cryptographic field: it
ties the statement to the target descriptor its predicate names. Until
this decision `Validate` read `subject[0].digest["sha256"]` and nothing
else, and `Covers` compared only that entry, in both the `attestation`
and `a2a` packages. in-toto's `DigestSet` is a map precisely so that a
subject can carry more than one algorithm, and a consumer whose policy
requires SHA-512 had no way to get one verified. Worse, a statement
carrying a correct `sha256` beside a wrong `sha512` was accepted: the
verifier never looked at the second entry, and a consumer that read it
would trust a digest nobody had checked.

## Decision

**The verifier knows two algorithms, `sha256` and `sha512`.** For each
one the subject carries, it recomputes the digest of the target
descriptor and refuses the statement if any does not match. At least one
known algorithm must be present: a subject with none, or with only an
algorithm the verifier does not know, is refused as a subject with no
digest was before. An unknown algorithm beside a known one is carried and
ignored, as in-toto allows. `Covers` holds on the same terms. The rule
lives in one internal package, `internal/subjectdigest`, so the MCP and
A2A verifiers cannot drift apart.

**What passmcp writes does not change.** `SubjectFor` keeps writing
`sha256` alone, and `graph.ServerID` keeps deriving a server's ID from
the `sha256` entry, so an ID does not change with the algorithms a
statement happens to carry.

**The schema does not change.** `digest` was already an object of
strings; which keys the verifier recomputes is a rule of `Validate`, not
of the schema, and `docs/verify.md` tells a schema-only consumer to apply
it.

## Consequences

- What `Validate` accepts changes in both directions, which this
  repository treats as breaking: a statement valid only by its `sha512`
  digest is now accepted, and one whose `sha512` does not match is now
  refused even when its `sha256` does. The change carries a tracking
  issue open for at least a week and a CHANGELOG entry that says so.
- The error for a subject with no usable digest now reads "no sha256 or
  sha512 digest", and a mismatch names the algorithm. A consumer that
  matched on the old strings must update; the strings were never API.
- A verifier older than this one refuses a statement that carries only
  `sha512`. A producer that wants old verifiers to accept it keeps
  writing `sha256`, as passmcp does.

## What would make this wrong

A third algorithm a consumer needs (SHA3, say). The list is one line in
`internal/subjectdigest`, but adding to it is again a change to what
`Validate` accepts and goes through the same notice.
