#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# scripts/readme-check.sh: the README follows the portfolio template. Each
# case writes a small README.md, go.mod and .github/demo.gif into the
# sandbox, breaks one thing the script checks, and expects it named.

load helpers

setup() {
  sandbox readme-check.sh
  mkdir -p "${SANDBOX}/.github"
  printf 'GIF89a' >"${SANDBOX}/.github/demo.gif"
  printf 'module example.com/demo\n\ngo 1.26.8\n' >"${SANDBOX}/go.mod"
  readme >"${SANDBOX}/README.md"
}

# badge prints one badge line of the row, with alt text $1 and image $2.
badge() {
  printf '  <a href="https://example.com"><img src="https://img.shields.io/%s" alt="%s" /></a>\n' "$2" "$1"
}

# readme prints a README for the project "demo" that passes the check.
readme() {
  cat <<'HEAD'
<h1 align="center">demo</h1>

<p align="center">
HEAD
  badge Build 'github/actions/workflow/status/o/demo/ci.yml?style=for-the-badge'
  badge Coverage 'endpoint?url=x&style=for-the-badge'
  badge Release 'github/v/release/o/demo?style=for-the-badge'
  badge Docs 'badge/go.dev-reference-007d9c?style=for-the-badge'
  badge 'OpenSSF Scorecard' 'ossf-scorecard/github.com/o/demo?style=for-the-badge'
  badge 'License: Apache-2.0' 'badge/license-Apache--2.0-blue.svg?style=for-the-badge'
  badge 'Go 1.26.8+' 'badge/go-1.26.8%2B-93450a.svg?style=for-the-badge&logo=go'
  cat <<'TAIL'
</p>

<p align="center">
  <img src=".github/demo.gif" alt="the demo verifying a statement" width="100%" />
</p>

## Contents
## Install
## Requirements
## Quick Start
## The demo ecosystem
## Capabilities at a glance
## Ecosystem comparison
## Benchmarks
## Features
## Configuration
## Examples
## When not to use demo
## Development
## Security
## Documentation
## Stability guarantees
## License
TAIL
}

# A code fence and an inline-code tick, kept out of single-quoted strings.
fence="\`\`\`"
tick="\`"

# edit applies a sed expression to the sandbox README.
edit() {
  sed -i.bak "$1" "${SANDBOX}/README.md"
}

check() {
  run "${SANDBOX}/scripts/readme-check.sh"
}

@test "a README that follows the template passes" {
  check
  [ "${status}" -eq 0 ]
  [[ "${output}" == *"follows the template (demo)"* ]]
}

@test "the repository's own README passes" {
  cp "${REPO_ROOT}/README.md" "${REPO_ROOT}/go.mod" "${SANDBOX}/"
  cp "${REPO_ROOT}/.github/demo.gif" "${SANDBOX}/.github/demo.gif"
  check
  [ "${status}" -eq 0 ]
  [[ "${output}" == *"follows the template (passmcp-reporting)"* ]]
}

@test "a heading out of order fails and prints the diff" {
  edit 's/^## Install$/## Tmp/; s/^## Requirements$/## Install/; s/^## Tmp$/## Requirements/'
  check
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"second-level headings differ from the template"* ]]
  [[ "${output}" == *"< Install"* ]]
  [[ "${output}" == *"> Install"* ]]
}

@test "a missing heading fails" {
  edit '/^## Benchmarks$/d'
  check
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"< Benchmarks"* ]]
}

@test "a heading inside a code fence does not count" {
  printf '\n%smd\n## Not a heading\n%s\n' "${fence}" "${fence}" >>"${SANDBOX}/README.md"
  check
  [ "${status}" -eq 0 ]
}

@test "an unresolved template variable outside code fails" {
  printf '\nBuilt by {{PROJECT_NAME}}.\n' >>"${SANDBOX}/README.md"
  check
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"unresolved template variables: {{PROJECT_NAME}}"* ]]
}

@test "a template variable inside a code fence or inline code passes" {
  printf '\n%stext\n{{PROJECT_NAME}}\n%s\n\nWrite %s{{PROJECT_NAME}}%s in the template.\n' \
    "${fence}" "${fence}" "${tick}" "${tick}" >>"${SANDBOX}/README.md"
  check
  [ "${status}" -eq 0 ]
}

@test "a badge missing from the row fails" {
  edit '/alt="Docs"/d'
  check
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"the badge row is not Build, Coverage"* ]]
  [[ "${output}" == *"< Docs"* ]]
}

@test "badges out of order fail" {
  edit 's/alt="Build"/alt="Tmp"/; s/alt="Coverage"/alt="Build"/; s/alt="Tmp"/alt="Coverage"/'
  check
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"the badge row is not Build, Coverage"* ]]
}

@test "a badge without style=for-the-badge fails" {
  edit '/alt="Release"/s/?style=for-the-badge/?style=flat/'
  check
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"every badge uses style=for-the-badge"* ]]
}

@test "a Go badge that disagrees with go.mod fails" {
  printf 'module example.com/demo\n\ngo 1.27.0\n' >"${SANDBOX}/go.mod"
  check
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"does not state go.mod's floor, 1.27.0"* ]]
}

@test "a README with no demo block fails" {
  edit '/demo\.gif/d'
  check
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"no demo block under the badge row"* ]]
}

@test "a missing .github/demo.gif fails" {
  rm "${SANDBOX}/.github/demo.gif"
  check
  [ "${status}" -eq 1 ]
  [[ "${output}" == *".github/demo.gif is missing"* ]]
}

@test "a README with no centred plain-text h1 fails" {
  edit 's|<h1 align="center">demo</h1>|# demo|'
  check
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"no centred plain-text"* ]]
}
