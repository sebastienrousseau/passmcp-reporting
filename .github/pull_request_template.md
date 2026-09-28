<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<!--
Thanks for contributing to passmcp-reporting.

Branch from `main` and target `main`. This package is imported by other
people's software, so a change to what Validate accepts or rejects is the
change to describe most carefully.
-->

## What this changes

<!-- One or two sentences. What is different after this merges? -->

## Why

<!-- The problem, not the patch. If it fixes an issue, link it: Fixes #123 -->

## How it was verified

- [ ] `make test` passes
- [ ] `make test-race` passes
- [ ] `make spec-verify` passes (required if a type or a json tag changed)
- [ ] A change to what a statement may carry is covered by a test that fails without it
- [ ] `CHANGELOG.md` has an entry under `## [Unreleased]`

## Risk

<!--
A statement this version accepts that the previous one rejected — or the
reverse — is the risk to name. A gateway on either side of the change sees
different admissions.
-->

---

- [ ] Commits are signed and carry a DCO `Signed-off-by` trailer (`git commit -s -S`)
