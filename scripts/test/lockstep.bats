#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# scripts/lockstep.sh: this repository's version against passmcp's latest
# release. curl is a stub that answers GitHub's API with CURL_CODE and
# CURL_BODY, and fails as the real one does under -f when the status is
# an error.

load helpers

setup() {
  sandbox lockstep.sh
  stub curl <<'STUB'
out="" fail=no
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out=$2; shift ;;
    -w|-H) shift ;;
    -*f*) fail=yes ;;
  esac
  shift
done
code=${CURL_CODE:-200}
if [ "${fail}" = yes ] && [ "${code}" -ge 400 ]; then
  echo "curl: (22) The requested URL returned error: ${code}" >&2
  exit 22
fi
printf '%s' "${CURL_BODY:-}" >"${out}"
printf '%s' "${code}"
STUB
}

# changelog writes a CHANGELOG.md whose newest release heading is $1.
changelog() {
  printf '# Changelog\n\n## [Unreleased]\n\n## [%s] — 2026-09-30\n' "$1" >"${SANDBOX}/CHANGELOG.md"
}

released() {
  export CURL_CODE=200 CURL_BODY="{\"tag_name\": \"v$1\"}"
}

@test "the first release passes before passmcp has any release" {
  changelog 0.0.1
  CURL_CODE=404 CURL_BODY='{"message": "Not Found"}' run "${SANDBOX}/scripts/lockstep.sh"
  [ "${status}" -eq 0 ]
  [[ "${output}" == *"0.0.1 is the release after passmcp's 0.0.0"* ]]
}

@test "only the first release passes before passmcp has any release" {
  changelog 0.0.2
  CURL_CODE=404 CURL_BODY='{"message": "Not Found"}' run "${SANDBOX}/scripts/lockstep.sh"
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"passmcp's latest release is 0.0.0"* ]]
}

@test "an HTTP error other than 404 fails and names the status" {
  changelog 0.0.5
  CURL_CODE=500 CURL_BODY='' run "${SANDBOX}/scripts/lockstep.sh"
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"returned HTTP 500"* ]]
}

@test "passmcp's own version passes" {
  changelog 0.0.4
  released 0.0.4
  run "${SANDBOX}/scripts/lockstep.sh"
  [ "${status}" -eq 0 ]
  [[ "${output}" == *"0.0.4 matches passmcp's latest release"* ]]
}

@test "the version after passmcp's passes, because this repository tags first" {
  changelog 0.0.5
  released 0.0.4
  run "${SANDBOX}/scripts/lockstep.sh"
  [ "${status}" -eq 0 ]
  [[ "${output}" == *"0.0.5 is the release after passmcp's 0.0.4"* ]]
}

@test "two versions ahead of passmcp fails" {
  changelog 0.0.6
  released 0.0.4
  run "${SANDBOX}/scripts/lockstep.sh"
  [ "${status}" -eq 1 ]
}

@test "behind passmcp fails" {
  changelog 0.0.3
  released 0.0.4
  run "${SANDBOX}/scripts/lockstep.sh"
  [ "${status}" -eq 1 ]
}
