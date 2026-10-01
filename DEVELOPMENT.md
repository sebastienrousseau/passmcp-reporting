<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Development

The single entry point for working on passmcp-reporting: toolchain, how to
reproduce every CI gate locally, and how a release is cut. If a gate fails
in CI and you cannot reproduce it from this file, that is a bug in this
file.

## Requirements

| Tool | Version | Why |
|---|---|---|
| Go | 1.26.8 or later, the `go` directive in `go.mod` | `GOTOOLCHAIN=auto` downloads it; CI tests on it and on latest stable, and never pins a version separately |
| make | any | Task runner for everything below |

Optional, only for the gate that uses it: `golangci-lint` (`make lint`),
`golangci-lint` and `jq` (`make complexity`),
`bats`, `jq` and `python3` (`make test-scripts`),
`markdownlint-cli2`, `codespell` and `lychee` (the Docs Lint workflow and
`pre-commit`), `curl` and `python3` (`make family`, `make lockstep`).

There is no dependency to download: `go.mod` has no `require` directive,
and keeping it that way is a rule, not a coincidence.

The Go tools CI runs besides the linter, `govulncheck` and `gorelease`,
are pinned in [`tools/go.mod`](tools/go.mod) with `tool` directives, a
module of its own outside the workspace so that nothing it requires
reaches the root module or the processor. `make tools` builds them into
`build/tools`; `make vulncheck` and `make api-check` use them. Dependabot
opens the bumps for `/tools` weekly. To bump by hand, or when Dependabot
does not offer one (gorelease lives in `golang.org/x/exp`, which has no
tagged releases, only pseudo-versions):

```sh
cd tools && GOWORK=off go get -tool golang.org/x/exp/cmd/gorelease@latest && GOWORK=off go mod tidy
cd tools && GOWORK=off go get -tool golang.org/x/vuln/cmd/govulncheck@latest && GOWORK=off go mod tidy
```

`golangci-lint` is pinned by version in `ci.yml` and in
`.devcontainer/post-create.sh`; change both together.

The README's toolchain badge states the same floor, and
`scripts/readme-check.sh` fails when the two disagree. A
[devcontainer](.devcontainer/devcontainer.json) boots to a working
`make build test` with the linter CI pins.

## Reproducing every CI gate

| CI job | Local command |
|---|---|
| Test (three OSes × two Go versions) | `make test` |
| Race & Shuffled Tests | `make test-race` |
| Coverage Gate (85% per package) | `make coverage` |
| Lint | `gofmt -l .`, `make lint` and `make complexity` |
| Vulnerability Scan | `make vulncheck`: the root module, and the processor with `GOWORK=off` |
| API Compatibility | `make api-check`, against the last root release before HEAD |
| Script Tests | `make test-scripts` (needs `bats`) |
| Repository Checks | `make apidoc spec-verify example-check release-versions family lockstep` |
| Licence Headers | `make spdx-check` |
| Markdown & Spelling | `make readme-check`, `markdownlint-cli2 '**/*.md'` and `codespell` |
| OpenSSF Scorecard (push to main, weekly) | not reproducible locally; results on [scorecard.dev](https://scorecard.dev/viewer/?uri=github.com/sebastienrousseau/passmcp-reporting) |
| Link Check | `lychee --offline --include-fragments '**/*.md'` |
| DCO check | `git log --format=%B origin/main.. \| grep Signed-off-by` |

`make` with no target runs the cheap gates in the order they fail fastest.

## Coverage

The gate is 85% statement coverage in every package with statements
(the example programs excepted: `make example-check` runs them). The
README's coverage badge is the root module's statement coverage measured
the same way: on every push to main the Manual workflow runs
`make coverage-badge`, which writes the shields.io endpoint document
`coverage.json` with `scripts/coveragebadge`, and deploys it with the
manual to <https://sebastienrousseau.com/passmcp-reporting/coverage.json>.
The badge is brightgreen at 90% and above, green from the 85% gate,
yellow from 70% and red below. The nested processor module is gated by
its own CI job and is not in this figure.

```sh
make coverage-badge   # writes coverage.json; the figure CI publishes
```

## Complexity

Every function is held to cyclomatic complexity 10, cognitive complexity
15 and 60 lines (with at most 50 statements), and every hand-written Go
file to 500 lines: the portfolio's ceilings. The function ceilings are
the `gocyclo`, `gocognit` and `funlen` settings in `.golangci.yml`; tests
are exempt, by the same file's exclusions. The functions already over
them when the ceilings were lowered are listed, with their measure, in
[`scripts/complexity-baseline.txt`](scripts/complexity-baseline.txt).
`make complexity` (`scripts/complexity.sh`, the Lint job's last step)
runs the three linters in the root module and the processor's and fails
on:

- a function or file over a ceiling that the baseline does not list;
- a listed one whose measure grew;
- a listed one that improved, or is now within its ceiling, while the
  baseline still says otherwise.

The baseline only shrinks. After bringing a function down, run
`scripts/complexity.sh --update` and commit the baseline with the change;
`--update` refuses to record a new or worse offender, and the fix for one
is to split the function, never an entry or a `//nolint`. The three
linters are not in `make lint`'s enabled set for that reason: there they
would fail on the listed backlog.

## Generated artefacts

| File | Generated by | Verified by |
|---|---|---|
| `spec/attestation/mcp-evaluation-v1.schema.json` | `make spec`, from the `attestation` types by reflection | `make spec-verify` in CI |
| `attestation/testdata/statement.json` | `PASSMCP_UPDATE_FIXTURES=1 go test ./attestation/` | `TestTheExampleFixtureIsCurrent` |
| `.github/demo.gif`, the README demo | `make demo`: [VHS](https://github.com/charmbracelet/vhs) records `.github/demo.tape` (needs `vhs`, `ttyd`, `ffmpeg`) | `make readme-check` checks it is present; look at it after regenerating |

Never edit one by hand. Change the source, regenerate, commit both. The demo
runs `examples/verify` on the fixtures, so re-render it when either changes
what that prints.

## Test layout

`attestation/attestation_test.go` is the contract: for every way a
statement can be wrong, a statement that is wrong that way and the error
`Validate` returns. A change to what is accepted starts there.
`diff_test.go` covers `Compare`. `spec/spec_test.go` checks the embedded
schema is the published one.

## Release model

The version is passmcp's. A release is cut when passmcp's is, on a
`feat/vX.Y.Z` branch, and because passmcp imports this module **this
repository tags first**:

1. Date the `## [X.Y.Z]` heading in `CHANGELOG.md` (`## [X.Y.Z] — date`),
   set `date-released` in `CITATION.cff`, and check the install snippets
   in `README.md` and the processor's README, the README's family
   sentence and `docs/releases/vX.Y.Z.md` name the version. The
   processor's highlights go in
   `docs/releases/agentgateway-extmcp/vX.Y.Z.md`. The two highlights files
   are the only hand-written part of the release pages.
2. `make lockstep` — the version is passmcp's latest release or the next.
3. `make release-versions` (`scripts/verify-release-versions.sh vX.Y.Z`).
4. Push a signed annotated tag `vX.Y.Z` with the message
   `passmcp-reporting vX.Y.Z`. The release workflow publishes the
   processor's image, then the release page; the module proxy serves the
   tag.
5. Tag the processor `integrations/agentgateway-extmcp/vX.Y.Z` with the
   message `agentgateway-extmcp vX.Y.Z`. Its `go.mod` requires a released
   root version; moving it to `vX.Y.Z`, when it needs the new verifier,
   is a commit after step 4
   ([ADR 0002](docs/adr/0002-processor-nested-module-via-go-work.md)).
   The tag's own run of the release workflow publishes the processor's
   page; it fails, and is re-run, if step 4's image is not pushed yet.
6. Read the tags and the release pages back from GitHub before calling
   it done.

The release pages are composed, never edited by hand.
`scripts/release-pages.sh` runs `scripts/releasepage` for each tag: the
title (`passmcp-reporting X.Y.Z`, or `agentgateway-extmcp X.Y.Z`, not
marked latest), the highlights, GitHub's generated `## What's Changed`
(and `## New Contributors` when there are any) from the previous tag of
the same series, a `## Checksums` section that says no files are attached
and names the digest ghcr.io serves for the processor image, and the
`**Full Changelog**` link. Each published page is read back and the step
fails unless GitHub shows what was composed. The workflow's
`workflow_dispatch` run prints both pages for the newest CHANGELOG
release and publishes nothing; locally (`gh` needs a token with contents
access for GitHub's generated notes):

```sh
scripts/release-pages.sh root X.Y.Z
scripts/release-pages.sh processor X.Y.Z
```

## Conventions

- Results and errors are values; nothing here logs or prints.
- Every exported identifier is documented, because pkg.go.dev is the
  manual.
- Anything in a statement is untrusted input.
