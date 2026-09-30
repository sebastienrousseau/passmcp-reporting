<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Contributing

passmcp-reporting is the attestation format passmcp writes and the Go verifier
that checks one. It is imported by software that makes decisions on what a
statement says, which is the thing to keep in mind for every change here.

## Getting started

1. Fork and clone the repository.
2. Install **Go** at the version the `go` directive in `go.mod` names, and
   **make**.
3. Create a branch from `main` and open the pull request against `main`.
   Every workflow filters on `pull_request: branches: [main]`, so a PR
   aimed elsewhere runs no CI.
4. Make the change.
5. Verify:

   ```bash
   make            # format, vet, lint, headers, schema, example, tests
   make test-race
   ```

## Commits

**Sign your commits cryptographically and add a DCO sign-off trailer.**
Both are required and both are enforced: signing proves who authored the
commit; the sign-off (`git commit -s`) certifies the
[Developer Certificate of Origin](https://developercertificate.org).
Merge commits are exempt from the DCO check.

Use [Conventional Commits](https://www.conventionalcommits.org/) with an
imperative subject: `feat(attestation): carry the plan's kernel`, not
`Added kernel`.

## What a change to the format needs

- **A test that fails without it.** A statement the change accepts or
  rejects that the previous version did not, in `attestation_test.go`.
- **`make spec`**, when a type or a `json` tag changed, and the
  regenerated schema in the same commit. CI fails when it drifts.
- **An entry under `## [Unreleased]` in `CHANGELOG.md`.**
- **A tracking issue, open for at least a week**, when the change alters
  what `Validate` accepts. A gateway on either side of such a change sees
  different admissions.
- **A bats test for a fix to a script.** A change to the behaviour of
  anything in `scripts/*.sh` comes with a case in `scripts/test/` that
  fails on the script before the change, run by `make test-scripts`.
  The helpers there stub the network and the clock, so a test never
  needs either.
- **No new dependency.** The package imports only the standard library so
  that a consumer can vet it in an afternoon; a pull request that adds a
  `require` line is declined on that ground alone.

## Pull request checklist

- [ ] `make` and `make test-race` pass
- [ ] The change is covered by a test that fails without it; for a script, a bats test
- [ ] `spec/` is regenerated if a type changed
- [ ] `CHANGELOG.md` has an entry under `## [Unreleased]`
- [ ] Commits are signed and carry a DCO sign-off
