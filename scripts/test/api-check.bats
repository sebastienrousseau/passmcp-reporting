#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# scripts/api-check.sh: which release it compares against, and which of
# gorelease's answers pass. gorelease is a stub that records its arguments
# and replays a report.

load helpers

setup() {
  sandbox api-check.sh
  stub gorelease <<'STUB'
printf '%s\n' "$@" >"${BATS_TEST_TMPDIR}/gorelease.args"
printf '%s\n' "${STUB_OUT:-# summary
Suggested version: v0.0.5}"
exit "${STUB_RC:-0}"
STUB
  export GORELEASE=${STUBS}/gorelease
}

base_used() {
  cat "${BATS_TEST_TMPDIR}/gorelease.args"
}

# The report gorelease printed for main at v0.0.4 while the module proxy
# still listed v0.0.3 as the newest release (CI run 36686509680).
readonly LAGGING="# summary
Cannot suggest a release version.
Can only suggest a release version when compared against the most recent version of this major: v0.0.3."

@test "with no release there is nothing to compare" {
  run "${SANDBOX}/scripts/api-check.sh"
  [ "${status}" -eq 0 ]
  [[ "${output}" == *"nothing to compare"* ]]
  [ ! -e "${BATS_TEST_TMPDIR}/gorelease.args" ]
}

@test "a branch after a release compares against that release" {
  git -C "${SANDBOX}" tag v0.0.3
  commit "release 0.0.4"
  git -C "${SANDBOX}" tag v0.0.4
  commit "work towards 0.0.5"
  run "${SANDBOX}/scripts/api-check.sh"
  [ "${status}" -eq 0 ]
  [ "$(base_used)" = "-base=v0.0.4" ]
}

@test "a tagged commit compares against the release before it, not itself" {
  git -C "${SANDBOX}" tag v0.0.3
  commit "release 0.0.4"
  git -C "${SANDBOX}" tag v0.0.4
  run "${SANDBOX}/scripts/api-check.sh"
  [ "${status}" -eq 0 ]
  [ "$(base_used)" = "-base=v0.0.3" ]
}

@test "the processor's nested tags are never the base" {
  git -C "${SANDBOX}" tag v0.0.3
  commit "processor release"
  git -C "${SANDBOX}" tag integrations/agentgateway-extmcp/v0.0.4
  commit "work"
  run "${SANDBOX}/scripts/api-check.sh"
  [ "${status}" -eq 0 ]
  [ "$(base_used)" = "-base=v0.0.3" ]
}

@test "a base the proxy does not list as newest is not an API finding" {
  git -C "${SANDBOX}" tag v0.0.3
  commit "release 0.0.4"
  git -C "${SANDBOX}" tag v0.0.4
  commit "work"
  STUB_OUT=${LAGGING} STUB_RC=1 run "${SANDBOX}/scripts/api-check.sh"
  [ "${status}" -eq 0 ]
  [[ "${output}" == *"no incompatible change against v0.0.4"* ]]
}

@test "an incompatible change fails" {
  git -C "${SANDBOX}" tag v0.0.4
  commit "work"
  STUB_OUT="# satellion.com/passmcp-reporting/attestation
## incompatible changes
ErrNoSuchCheck: removed" STUB_RC=0 run "${SANDBOX}/scripts/api-check.sh"
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"changed incompatibly against v0.0.4"* ]]
}

@test "an incompatible change fails even when no version is suggested" {
  git -C "${SANDBOX}" tag v0.0.4
  commit "work"
  STUB_OUT="## incompatible changes
ErrNoSuchCheck: removed
${LAGGING}" STUB_RC=1 run "${SANDBOX}/scripts/api-check.sh"
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"changed incompatibly"* ]]
}

@test "a gorelease that could not load the packages fails" {
  git -C "${SANDBOX}" tag v0.0.4
  commit "work"
  STUB_OUT="# summary
Cannot suggest a release version.
Errors were found." STUB_RC=1 run "${SANDBOX}/scripts/api-check.sh"
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"did not complete against v0.0.4"* ]]
}

@test "a diagnostic beside the suggestion refusal still fails" {
  git -C "${SANDBOX}" tag v0.0.4
  commit "work"
  STUB_OUT="# diagnostics
go.mod: requirement on a retracted version
${LAGGING}" STUB_RC=1 run "${SANDBOX}/scripts/api-check.sh"
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"did not complete"* ]]
}
