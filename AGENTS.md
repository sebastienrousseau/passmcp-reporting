<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Working on passmcp-reporting as an AI agent

Invariants for AI-assisted contributions: the constraints a plausible
change breaks for a reason the code does not state. Read
[DEVELOPMENT.md](DEVELOPMENT.md) for the toolchain and the local form of
every CI gate.

## Hard gates

| Gate | Command |
|---|---|
| 85% statement coverage, every package with statements | `go test ./... -cover` |
| Race detector, randomised order | `make test-race` |
| Lint at zero findings | `make lint` |
| SPDX header on every source file | `make spdx-check` |
| The schema matches the types | `make spec-verify` |
| The example verifies the fixture | `make example-check` |
| The family manifest's row is true | `make family` |
| The version is passmcp's, or the next one | `make lockstep` |

## Commits

- **Every commit must be cryptographically signed and carry a DCO
  `Signed-off-by` trailer.** An agent's shell usually cannot reach the
  maintainer's ssh-agent; hand the commits over as a script rather than
  producing unsigned history that has to be rewritten.
- Conventional Commits for the subject line.
- Never rewrite published history.

## Versioning

- **The version is passmcp's.** Never choose one here. It lives in the
  newest `## [x.y.z]` heading in `CHANGELOG.md` and nowhere else.
- passmcp imports this module, so **this repository tags first** for a
  release; `scripts/lockstep.sh` allows being exactly one patch ahead
  of passmcp's latest release for that reason, and nothing else.

## Things that look like bugs and are not

- **The subject digest is over a target descriptor, not an artifact.**
  `predicate.subjectKind` says so. A change that digests the statement's
  bytes, or the report, breaks every consumer's `Covers`.
- **`Validate` refuses a score without a rubric version.** A number that
  cannot be recomputed is not a claim. Do not make the field optional to
  accept a producer that omits it; the producer is wrong.
- **The rubric is not here.** It is generated from passmcp's scorer, in
  passmcp. A copy here would drift the day the scorer changed.
- **`spec/` is generated.** Edit the types and run `make spec`; never
  the JSON.

## Things that are load-bearing

- **Standard library only.** The package has no `require` directive and
  must keep none. That is the whole reason it is a separate module: a
  consumer adding a verifier must not inherit a dependency graph.
- **Everything in a statement is untrusted.** A statement can come from
  anyone. Nothing here reads a file, opens a connection or follows a
  reference on a statement's say-so.
- **What `Validate` accepts is the public API.** A statement this
  version rejects and the last accepted, or the reverse, is a breaking
  change whatever the signatures did, and needs a tracking issue and a
  changelog entry that says so.

## Scope

- Do not couple a structure or documentation cleanup to a behaviour
  change.
- Do not add a CI gate that does not currently pass.
