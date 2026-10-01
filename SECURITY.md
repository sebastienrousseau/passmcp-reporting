<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Security Policy

passmcp-reporting is a Go library that reads and verifies attestation
statements other software will make admission decisions on. Its security
posture is about one thing: a statement that `Validate` accepts says what
it appears to say.

## Reporting a Vulnerability

Report security issues through [GitHub's private vulnerability reporting](https://github.com/sebastienrousseau/passmcp-reporting/security/advisories/new). Do not open a public issue.

You will receive an acknowledgement within **72 hours**. A confirmed
vulnerability is fixed and released within **90 days** of the report, or
sooner when a fix is straightforward; if the window cannot be met you will
be told why and given a revised date.

## Supported Versions

Only the latest release is supported. The version is passmcp's; see the
lockstep rule in [CHANGELOG.md](CHANGELOG.md).

## Security Measures

Each item names the test that enforces it, so the claim can be checked
rather than taken on trust. The full argument, with the threat model,
the trust boundaries of the verifier and of the processor, and the
latest security review, is [docs/security-model.md](docs/security-model.md).

- **The subject digest is recomputed, never trusted.** `Validate` derives
  the digest from the predicate's target descriptor and compares it with
  the subject's; a statement whose displayed subject was rewritten to
  name another server fails. `attestation_test.go` covers the tampered
  cases: renamed subject, altered digest, a second subject, a mismatched
  predicate type.
- **A score without its rubric is invalid.** A number nobody can
  recompute is not a claim; `Validate` refuses it.
- **The verifier has no network, no files and no dependencies.** The
  `attestation` and `a2a` packages import only the standard library and
  touch nothing outside the bytes they are given, so a hostile statement
  has no side channel to reach. The root module's `go.mod` has no
  `require` directive. Two parts of the repository do more, and are
  held to more: `graph`'s `Load` and `Save` read and write the one store
  file in the directory the caller names, and the agentgateway processor
  under `integrations/` is a network service with dependencies (gRPC,
  protobuf) that reads files and fetches statements over https. CI runs
  `govulncheck` on both modules on every push.
- **The schema is derived from the types**, by `scripts/specgen`, and CI
  fails when the committed file drifts, so a consumer validating by
  schema and one validating by this package agree.
- **Releases are signed.** Each release tag is an annotated tag signed
  with the maintainer's SSH key, and the processor image carries SLSA
  build provenance attested to its digest.
  [docs/signing.md](docs/signing.md) shows how to verify both.
- **Signatures are out of scope here.** This package checks the
  statement's structure and integrity. Whether the bytes came from who
  they claim is the job of the envelope around them — DSSE, cosign,
  `gh attestation verify` — and passmcp's manual describes how to sign a
  statement and check the signature. A consumer must do both.
