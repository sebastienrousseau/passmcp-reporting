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

## Code review

**Who reviews.** The maintainer, Sebastien Rousseau
(`@sebastienrousseau`), reviews every pull request; `CODEOWNERS`
requests the review automatically. The project has one maintainer, so
there is no second reviewer: the maintainer's own changes are held to
the same checklist below and to the CI gates, which `main`'s branch
protection requires to pass before anything merges, but no second
person approves them. That follows from the single-maintainer model
[GOVERNANCE.md](GOVERNANCE.md) states, and the
[security model](docs/security-model.md#7-continuity-and-bus-factor)
lists what compensates for it.

**The process.**

1. A pull request targets `main`; the PR Base check fails any other
   base.
2. CI runs: tests on three operating systems and two Go versions, the
   race detector, the 85% coverage gate, lint, `govulncheck`, the API
   compatibility check, the repository checks, the script tests,
   licence headers, Markdown, spelling and links, and the DCO check.
   A red gate is fixed in the pull request, never waived.
3. The maintainer reads the whole diff against the list below and
   either approves, asks for changes in review comments, or explains
   why the change is declined. Expect a first response within a week
   ([SUPPORT.md](SUPPORT.md)).
4. Accepted work is merged by the maintainer. Between releases it is
   collected on the release branch `feat/vX.Y.Z`, whose single pull
   request into `main` is the release; a contribution may be carried
   there as commits, with its own pull request closed with a link to
   the release pull request. Dependabot updates arrive the same way.

**What the reviewer checks.**

- **Tests.** The change is covered by a test that fails without it: a
  statement the change accepts or rejects in `attestation_test.go` (or
  the `a2a` and `graph` equivalents), a processor case, or a bats test
  in `scripts/test/` for a script. Coverage stays at or above 85% in
  every package.
- **Security.** Anything read from a statement, a request or a fetched
  source is treated as untrusted; nothing new reads a file, opens a
  connection or follows a reference on a statement's say-so; nothing
  new can put a credential or unbounded text in a denial or a log
  line. A change to a trust boundary updates
  [docs/security-model.md](docs/security-model.md).
- **SPDX.** Every new source file carries an SPDX header (`make
  spdx-check`); copied files keep their upstream attribution.
- **CHANGELOG.** A user-visible change has an entry under
  `## [Unreleased]`.
- **API compatibility.** `make api-check` passes, and a change to what
  `Validate` accepts, in either direction, is treated as breaking: it
  has its tracking issue, open for at least a week, and a CHANGELOG
  entry that says so. `spec/` is regenerated in the same commit when a
  type changed.
- **Dependencies.** The root module gains no `require` line; a new
  dependency of the processor says why in its commit.
- **Commits.** Signed, with a DCO sign-off, Conventional Commit
  subjects, and one concern per commit.

**What makes a change acceptable.** Every CI gate is green, every item
above holds, the change fits the project's scope (the format, its
verifier and the processor; not verdicts, which are passmcp's), and a
behaviour change is not bundled with an unrelated cleanup. A change
that meets all of that is merged; one that does not is sent back with
the reason.

## Pull request checklist

- [ ] `make` and `make test-race` pass
- [ ] The change is covered by a test that fails without it; for a script, a bats test
- [ ] `spec/` is regenerated if a type changed
- [ ] `CHANGELOG.md` has an entry under `## [Unreleased]`
- [ ] Commits are signed and carry a DCO sign-off
