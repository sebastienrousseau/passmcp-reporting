<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Architecture Decision Records

Decisions about this repository that will be questioned later, with the
reasoning that produced them. Records are immutable once merged; a
decision that changes gets a new record superseding the old one.

Decisions about the *format itself* — what a verdict means, how the score
is derived, why the format is Apache-2.0 — are passmcp's, in
[passmcp's ADRs](https://github.com/sebastienrousseau/passmcp/blob/main/docs/adr/README.md).
This directory records what is decided here.

| # | Decision | Status |
|---|---|---|
| [0001](0001-verifier-in-its-own-module.md) | The verifier is its own module; the rubric and the renderers stay in passmcp until a consumer needs them | Accepted |
| [0002](0002-processor-nested-module-via-go-work.md) | The agentgateway processor is a nested module, built from the tree through `go.work` and installed against the tagged release | Accepted |
| [0003](0003-subject-digest-sha256-or-sha512.md) | The verifier accepts a subject digest in sha256, sha512 or both, recomputing every one present; passmcp keeps writing sha256 | Accepted |
