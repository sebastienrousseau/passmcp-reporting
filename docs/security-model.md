<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Security model and assurance case

**Status:** living document. **Last security review:** 2026-09-30, below.
**Owner:** Sebastien Rousseau ([@sebastienrousseau](https://github.com/sebastienrousseau)).
**Scope:** the root module `satellion.com/passmcp-reporting` (the
`attestation`, `a2a`, `graph` and `spec` packages), the nested module
`integrations/agentgateway-extmcp` (the agentgateway processor and its
container image), and the pipeline that releases them.

This is the argument that passmcp-reporting is secure for what it is
used for, with the evidence for each claim: a test, a CI gate, or a line
of code a reviewer can read. [SECURITY.md](https://github.com/sebastienrousseau/passmcp-reporting/blob/main/SECURITY.md)
is how to report a vulnerability; this is why the design should not
have many.

## 1. What is being protected

Software on the other side of this repository makes admission
decisions: a gateway routes to an MCP server or refuses to, a registry
displays a grade, a pipeline fails a build. The asset is **the
correctness of those decisions**:

- A statement the verifier accepts says what it appears to say, about
  the server it names.
- A processor denies what its policy denies, and says nothing through a
  denial that it should not.
- A release is what the maintainer built.

It is not the confidentiality of a statement. A statement is a claim
meant to be shown; it holds no credential, and the `graph` model has no
field one could be kept in.

## 2. Two components, two trust boundaries

The repository ships two very different things, and the model treats
them separately.

### The verifier library (root module)

`attestation`, `a2a` and `graph` are pure functions over bytes the
caller supplies, plus, in `graph`, one store file. They import only the
standard library; `go.mod` has no `require` directive.

| # | Boundary | What crosses it | Trusted? |
|---|---|---|---|
| L1 | caller → `Parse` / `Validate` | statement bytes | **No.** A statement can come from anyone. |
| L2 | `Validate` → caller | accept or refuse, with the reasons | The caller trusts the verdict on structure and integrity only. |
| L3 | caller → `graph.Load` / `graph.Save` | a directory path, and the `graph.json` in it | The path is the caller's; the file's contents are parsed as untrusted. |

What the library does **not** do is part of its security: it never
reads a file (outside `graph`'s store), opens a connection, follows a
reference in a statement, or checks a signature. Who wrote a statement
is the job of the envelope around it (DSSE, cosign, `gh attestation`),
and a consumer must verify that as well.

### The agentgateway processor (nested module)

`agentgateway-extmcp` is a long-running gRPC server with dependencies
(gRPC, protobuf), that reads files and fetches over https.

| # | Boundary | What crosses it | Trusted? |
|---|---|---|---|
| P1 | operator → processor | flags, the configuration file, TLS key pair | **Yes.** The operator chooses what is gated and where statements come from. |
| P2 | processor → attestation source | a file read, or an https GET to a URL in the configuration | The **location** is the operator's; the **bytes** are untrusted and verified. |
| P3 | agentgateway → processor | gRPC `CheckRequest` / `CheckResponse`: backend names, method, headers, parameters | The gateway is trusted to name the backends; anyone who can reach the port can ask. |
| P4 | processor → MCP client, through the gateway | the decision and, on a denial, its reason | Output to untrusted clients: it must not carry secrets or grow without bound. |
| P5 | processor → operator's logs | JSON lines on stderr | The operator's own channel. |

The processor fetches statements when it starts and when it reloads,
never on the request path, and it never fetches a URL that came from a
statement or from a request.

### The release (both modules)

| # | Boundary | What crosses it |
|---|---|---|
| R1 | maintainer → git | signed commits, and signed annotated release tags |
| R2 | CI → registries | the Go module proxy serving the tag; the processor image on ghcr.io with SLSA provenance |
| R3 | Dependabot and contributors → repository | pull requests, reviewed by the maintainer ([CONTRIBUTING.md](https://github.com/sebastienrousseau/passmcp-reporting/blob/main/CONTRIBUTING.md#code-review)) |

## 3. Threat model

| Threat | Actor | Against | Where it is countered |
|---|---|---|---|
| A statement edited after it was made, to name another server or show another score | anyone who handles the file | L1 | C1 |
| A statement crafted to crash, stall or exhaust the verifier | anyone | L1 | C2 |
| A statement of one kind read as another (A2A as MCP) | anyone | L1 | C1 |
| A statement fetched in the clear and swapped in transit | network attacker | P2 | C4 |
| The processor made to fetch an address of the attacker's choosing | a statement author, an MCP client | P2 | C4 |
| The gateway's traffic to the processor read or altered | network attacker | P3 | C5 |
| A credential, or unbounded attacker text, delivered to MCP clients in a denial | whoever controls a statement or its origin | P4 | C6 |
| A processor that has not loaded its configuration answering "pass" | fault, not attacker | P3 | C3 |
| A tampered release: tag, module or image | registry or account compromise | R1, R2 | C7 |
| A vulnerable dependency | upstream | P1–P5 | C8 |

**Out of scope.** Whether a statement's author is who they claim to be
(the envelope's job, above); whether passmcp judged a server correctly
(passmcp's); a malicious operator, who configures the processor and can
make it say anything; a compromised gateway, which already sees every
call; denial of service by a client that can reach the processor's port
beyond gRPC's own limits (restrict who can reach it); and side channels
in the Go runtime.

## 4. Claims and evidence

### C1. A statement `Validate` accepts is intact and about the server it names

`Validate` recomputes the subject digest from the predicate's target
descriptor and compares it with the subject's, in every known algorithm
(`sha256`, `sha512`) the subject carries, so an edited endpoint,
transport or subject fails; the subject's name must be the target's
endpoint, because that is what in-toto tooling displays; there must be
exactly one subject; the statement and predicate types must be this
format's; and every verdict status must come from the closed set.
`Covers` answers "is this about that server" by recomputing the digest,
never by comparing strings.

- Code: `attestation/verify.go` (`Validate`, `validateSubject`,
  `Covers`), `a2a/verify.go`.
- Evidence: `TestValidateNamesEveryProblem`,
  `TestCoversRecomputesRatherThanCompares`,
  `TestSubjectForIsStableAndDistinct` (attestation);
  `TestValidateRefusesEachBrokenRule` and
  `TestThePredicateTypesDoNotCross` (a2a: neither verifier accepts the
  other's statement, and an agent and a server at one URL have different
  digests); the processor's `TestCheckRequest` denies the "tampered"
  statement.

### C2. Hostile bytes cannot crash the parsers or make them do work out of proportion

Parsing is `encoding/json` into typed structs, then linear passes over
the verdicts; nothing recurses on attacker-controlled depth beyond the
standard decoder's own limit, and nothing is interpreted.

- Evidence: `attestation.FuzzParse`, `a2a.FuzzParse` and
  `graph.FuzzParse` check that `Parse` never panics and that anything it
  accepts round-trips through `Marshal` byte for byte and still covers
  its own target. Their seeds run in every `go test`.
- The library does not cap input size: it is a function over bytes the
  caller already holds, and the right limit depends on the caller. A
  consumer reading statements from the network must bound the read
  first, as the processor does (C6).

### C3. The processor fails closed

A target with no usable statement is denied unless the policy sets
`requireAttestation: false`, which defaults to true. A processor with no
configuration loaded answers gRPC `UNAVAILABLE` rather than a decision,
so the gateway's `failureMode` chooses, and a reload that fails keeps
the last good table instead of emptying it. A configuration with an
unknown field, or an `http://` attestation, is refused.

- Code: `processor/config.go` (`Policy.Requires`, `Validate`,
  `DisallowUnknownFields`), `processor/server.go` (`CheckRequest`),
  `cmd/agentgateway-extmcp/main.go` (`reloadOn`).
- Evidence: `TestCheckRequest` (unknown, unusable, below-score and
  failing-category targets denied), `TestUnloadedStoreIsAnErrorNotADecision`,
  `TestReload`, `TestParseConfigNamesEveryProblem`.

### C4. Statements are fetched only from where the operator said, over https

`Loader.Load` accepts a file path or an `https://` URL and refuses every
other scheme. Its only caller passes the `attestation` field of the
operator's configuration; no statement field and no request field is
ever fetched, which is what rules out server-side request forgery by a
statement author or an MCP client. A redirect is followed only to
another `https://` URL, and at most ten times. Certificates are checked
by the standard library; nothing sets `InsecureSkipVerify`.

- Code: `processor/load.go` (`Load`, `client`, `fetch`),
  `processor/store.go` (`verify`).
- Evidence: `TestLoaderReadsFilesAndHTTPS` ("http refused", "other
  scheme"), `TestLoaderRefusesARedirectOffHTTPS`,
  `TestParseConfigNamesEveryProblem` ("file path or an https").
- Residual: an https origin can still redirect to any https host,
  including an internal one. The origin is the operator's choice, and a
  statement fetched from anywhere is still verified before it counts.

### C5. The gateway's traffic to the processor can be encrypted

With `-tls-cert` and `-tls-key` the listener serves TLS 1.2 or 1.3
only; naming one without the other, or a key that does not match, stops
the process before it listens. Without them it is plaintext and, by
default, bound to `127.0.0.1`.

- Code: `cmd/agentgateway-extmcp/tls.go`.
- Evidence: `TestRunServesOverTLS`, `TestTLSConfigFloorIsTLS12`,
  `TestTLSListenerRefusesOldVersionsAndPlaintext`,
  `TestServerTLSRefusesAnIncompletePair`,
  `TestRunRefusesAHalfTLSConfigurationBeforeListening`.
- Residual: the processor does not authenticate its clients (no mutual
  TLS). Its answers carry a decision and a score, no secret. The
  container image listens on `0.0.0.0` in plaintext unless given a key
  pair, because a container's loopback is unreachable from the gateway;
  its README says to add TLS or restrict the network.

### C6. A denial carries no credential and is bounded

The reason an attestation is unusable reaches every MCP client whose
call to that target is denied. A failed fetch reports the cause without
the URL, whose query may be a presigned credential; the reason is cut
at 512 bytes, since it can quote a statement; statements are capped at
`-max-bytes` (1 MiB by default) and each fetch at `-fetch-timeout` (10 s).
Logs are JSON lines from `log/slog`, which escapes what it writes.

- Code: `processor/load.go` (`fetch`, `capped`), `processor/store.go`
  (`bounded`).
- Evidence: `TestAFailedFetchDoesNotRepeatTheURL`,
  `TestAnUnusableReasonIsBounded`, `TestLoaderReadsFilesAndHTTPS`
  ("https too large", "file too large"), `TestLoaderDefaultsAreApplied`.

### C7. A release is what the maintainer built

Release tags are annotated and signed with the maintainer's SSH key.
The Go checksum database pins each module version's bytes. The
processor image is built by the release workflow, pushed by digest, and
carries SLSA build provenance signed through GitHub's artifact
attestations. Every workflow declares its permissions (all but the
DCO check start from `permissions: {}`, and that one reads contents
only), each job is granted only what it uses, and every action is
pinned by commit SHA.

- Evidence: [Verifying a release](signing.md), every command run
  against v0.0.4; `.github/workflows/release.yml`; OpenSSF Scorecard on
  every push to `main`.

### C8. Dependencies are few, watched and scanned

The root module has none. The processor's are gRPC, protobuf and their
`golang.org/x` modules. CI runs `govulncheck` on both modules on every
pull request; Dependabot watches both modules, the processor's image
bases and the workflow actions; CodeQL analyses the Go code and the
workflows on every pull request.

- Evidence: `.github/workflows/ci.yml` (Vulnerability Scan),
  `.github/dependabot.yml`, `.github/workflows/codeql.yml`.

## 5. Secure design principles applied

| Principle | Where |
|---|---|
| Economy of mechanism | The verifier is standard library only and under a thousand lines per package; a reviewer can read all of it ([ADR 0001](adr/0001-verifier-in-its-own-module.md)). |
| Fail-safe defaults | `requireAttestation` defaults to true; an unloaded store is an error, not a pass; an unknown predicate type, transport, status or configuration field is refused; half a TLS key pair is refused rather than read as plaintext. |
| Complete mediation | `CheckRequest` decides every backend a request names, from the current snapshot, every time; the first refusal denies the call. |
| Open design | The format, its schema and this document are public; nothing depends on an attacker not knowing how it works. |
| Least privilege | The processor image runs as a non-root user on a distroless base; each CI job holds only the permissions it names; the processor reads its sources at load time, not per request. |
| Least common mechanism | The processor is its own module, so its dependencies never reach a verifier consumer's build ([ADR 0002](adr/0002-processor-nested-module-via-go-work.md)). |
| Separation of duties | Integrity of a statement's contents (this library) and authenticity of its author (the envelope) are checked by different code, and both are required. |
| Input validation by allowlist | Closed sets for statement types, transports, verdict statuses and URL schemes; `DisallowUnknownFields` on the processor's configuration. |
| Defence in depth | The subject digest and the subject name are both checked; the schema, generated from the types, gives a non-Go consumer the same structural check. |

## 6. Common weaknesses and how they are countered

| CWE | Weakness | Counter | Test |
|---|---|---|---|
| CWE-20 | Improper input validation | `Validate` refuses each broken rule; the processor's configuration is validated in full | `TestValidateNamesEveryProblem`, `TestValidateRefusesEachBrokenRule`, `TestValidateRefusesABrokenGraph`, `TestParseConfigNamesEveryProblem`, the three `FuzzParse` |
| CWE-345 | Insufficient verification of data authenticity | Subject digest recomputed from the target; name checked; signature left to the envelope and documented as required | `TestCoversRecomputesRatherThanCompares`, `TestCheckRequest` ("tampered") |
| CWE-400, CWE-770 | Uncontrolled resource consumption | Statement size cap, fetch timeout, bounded denial reasons, loading off the request path | `TestLoaderReadsFilesAndHTTPS`, `TestLoaderDefaultsAreApplied`, `TestAnUnusableReasonIsBounded` |
| CWE-918 | Server-side request forgery | Only the operator's configured source is fetched; redirects must stay on https | `TestLoaderRefusesARedirectOffHTTPS`, `TestLoaderReadsFilesAndHTTPS` |
| CWE-319 | Cleartext transmission | `http://` sources and redirects to them refused; TLS for the gRPC listener | `TestLoaderReadsFilesAndHTTPS`, `TestRunServesOverTLS`, `TestTLSListenerRefusesOldVersionsAndPlaintext` |
| CWE-326, CWE-327 | Weak TLS or hashing | TLS 1.2 floor; SHA-256 or SHA-512 subject digests, every one present recomputed | `TestTLSConfigFloorIsTLS12`, `TestSubjectForIsStableAndDistinct`, `TestSubjectDigestAlgorithms` |
| CWE-295 | Improper certificate validation | Standard library verification; no `InsecureSkipVerify` anywhere in the tree | `TestLoaderReadsFilesAndHTTPS` (only the test server's own CA is trusted) |
| CWE-209, CWE-532 | Sensitive data in errors or logs | Fetch errors drop the URL; the graph model cannot hold a secret | `TestAFailedFetchDoesNotRepeatTheURL`, `TestTheModelHasNoFieldForASecret` |
| CWE-362 | Race condition | Requests read one immutable snapshot, swapped under a lock; the race detector runs in CI | `make test-race`; the processor's CI job runs `-race` |
| CWE-1104 | Unmaintained third-party components | No dependency in the root module; `govulncheck` and Dependabot on the processor | Vulnerability Scan job |

## 7. Continuity and bus factor

The project has one maintainer, which is a bus factor of one, recorded
here and in [GOVERNANCE.md](https://github.com/sebastienrousseau/passmcp-reporting/blob/main/GOVERNANCE.md)
rather than hidden. The compensating controls: every change goes
through a pull request against a CI gate that does not depend on the
maintainer's judgement (tests at an 85% coverage floor, the race
detector, lint, `govulncheck`, API compatibility, the generated schema);
the succession procedure is passmcp's, which this repository follows;
and Apache-2.0 lets anyone fork without permission.

## 8. Security review — 2026-09-30

Performed by the maintainer with AI assistance (Claude Code), against
this document, at the tip of `feat/v0.0.5`.

**Method.** Every non-generated Go file in both modules read against
the threat model above; the CI, release and Dependabot configuration
read for the release boundary; `go test -race` on both modules;
`golangci-lint` with `gosec`; `govulncheck` on both modules (no
vulnerabilities found); each `FuzzParse` run for 60 seconds (nothing
found); every command in [Verifying a release](signing.md) run against
v0.0.4.

**Findings.**

Fixes are named by their commit subject on the `feat/v0.0.5` branch.

| ID | Severity | Finding | Status |
|---|---|---|---|
| SR-1 | Medium | The processor followed a redirect from an https attestation URL to `http://`, fetching the statement in the clear although `http://` sources are refused | Fixed: redirects must stay on https (`fix: refuse attestation redirects off https`) |
| SR-2 | Medium | A failed fetch put the full attestation URL into the denial reason sent to MCP clients; a presigned URL's query is a credential | Fixed (`fix: keep URLs and long quotes out of denials`) |
| SR-3 | Low | The denial reason could quote up to `-max-bytes` of statement text in every denial and log line | Fixed: cut at 512 bytes, same commit |
| SR-4 | Medium | The processor's gRPC listener had no TLS | Fixed: `-tls-cert` and `-tls-key`, TLS 1.2 floor (`feat: serve the processor's gRPC over TLS`) |
| SR-5 | Low | SECURITY.md said "no network, no files, no dependencies" of the project; that is true of the verifier packages only, not of `graph`'s store or the processor | Fixed in SECURITY.md with this document |
| SR-6 | Low | `a2a.Parse` had no fuzz target, unlike the other two parsers | Fixed (`test: fuzz the A2A statement parser`) |
| SR-7 | Low | Dependabot watched only the root module, which has no dependencies, and not the processor's module or its image bases | Fixed (`ci: watch the processor's module and image bases`) |
| SR-8 | Info | The container image listens in plaintext on all interfaces unless given a key pair | Accepted: a container's loopback is unreachable from the gateway; documented in the processor's README |
| SR-9 | Info | `attestation.Parse`, `a2a.Parse` and `graph.Parse` do not cap input size | Accepted by design (C2): the caller holds the bytes and sets the limit; the processor does |
| SR-10 | Medium | The maintainer's GitHub account lists eight SSH signing keys, one titled `draft-tap-bot`; a tag signed by any of them verifies against the published list | Open, maintainer action: review the account's signing keys. [Verifying a release](signing.md) names the one key that has signed every release and shows how to accept only it |
| SR-11 | Low | CI runs `gorelease@latest` and `govulncheck@latest`, unpinned | Open, maintainer decision: both jobs hold only `contents: read` and no secret, so a compromised tool can fail a check but not publish; pinning trades that for update churn |
| SR-12 | Info | The processor does not authenticate the gateway (no mutual TLS) | Accepted: its answers hold no secret; restrict the port with the network |

## 9. Keeping this current

This document changes with any change to a boundary in section 2: a new
input, a new output channel, a new dependency or a new release
artefact. The next review is due with the first of: a change of that
kind, a reported vulnerability, or 2027-03-31.
