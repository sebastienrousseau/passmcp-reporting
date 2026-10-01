#!/usr/bin/env bats
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# scripts/release-pages.sh: what it hands scripts/releasepage, and how it
# waits for the processor image. curl is a stub for ghcr.io whose manifest
# answers with a digest from the DIGEST_AFTERth request on ("never" for
# none), sleep records its argument instead of sleeping, and go prints the
# arguments it would have run releasepage with.

load helpers

readonly NESTED=integrations/agentgateway-extmcp

setup() {
  command -v jq >/dev/null || skip "jq is not installed"
  sandbox release-pages.sh
  stub curl <<'STUB'
out=""
args=("$@")
for i in "${!args[@]}"; do
  [ "${args[$i]}" = -o ] && out=${args[$((i + 1))]}
done
case "$*" in
  *ghcr.io/token*) echo '{"token": "stub"}' >"${out}" ;;
  *ghcr.io/v2/*)
    n=$(($(cat "${BATS_TEST_TMPDIR}/manifest.count" 2>/dev/null || echo 0) + 1))
    echo "${n}" >"${BATS_TEST_TMPDIR}/manifest.count"
    after=${DIGEST_AFTER:-1}
    if [ "${after}" = never ] || [ "${n}" -lt "${after}" ]; then exit 22; fi
    printf 'HTTP/2 200\r\ndocker-content-digest: sha256:feed\r\n' ;;
  *) exit 2 ;;
esac
STUB
  stub sleep <<'STUB'
echo "$1" >>"${BATS_TEST_TMPDIR}/sleeps"
STUB
  stub go <<'STUB'
printf '%s\n' "$@"
STUB
  export RELEASE_PAGES_WAIT=90 RELEASE_PAGES_POLL=30
}

sleeps() {
  if [ -f "${BATS_TEST_TMPDIR}/sleeps" ]; then wc -l <"${BATS_TEST_TMPDIR}/sleeps" | tr -d ' '; else echo 0; fi
}

@test "--publish waits for an image the other release run is still pushing" {
  DIGEST_AFTER=3 run "${SANDBOX}/scripts/release-pages.sh" --publish processor 0.0.4
  [ "${status}" -eq 0 ]
  [[ "${output}" == *"waiting for ghcr.io/"* ]]
  [[ "${output}" == *"is \`sha256:feed\`"* ]]
  [[ "${output}" == *"-publish"* ]]
  [ "$(sleeps)" -eq 2 ]
}

@test "--publish fails once the wait runs out, and says why" {
  DIGEST_AFTER=never run "${SANDBOX}/scripts/release-pages.sh" --publish processor 0.0.4
  [ "${status}" -eq 1 ]
  [[ "${output}" == *"passmcp-agentgateway-extmcp:0.0.4 is not published"* ]]
  [ "$(sleeps)" -eq 3 ]
}

@test "--publish names an image that is already there without waiting" {
  DIGEST_AFTER=1 run "${SANDBOX}/scripts/release-pages.sh" --publish root 0.0.4
  [ "${status}" -eq 0 ]
  [[ "${output}" == *"is \`sha256:feed\`"* ]]
  [ "$(sleeps)" -eq 0 ]
}

@test "a dry run names an unpublished image as such and does not wait" {
  DIGEST_AFTER=never run "${SANDBOX}/scripts/release-pages.sh" processor 0.0.4
  [ "${status}" -eq 0 ]
  [[ "${output}" == *"sha256:<not published yet>"* ]]
  [[ "${output}" != *"-publish"* ]]
  [ "$(sleeps)" -eq 0 ]
}

@test "the change list starts at the previous tag of the same series" {
  git -C "${SANDBOX}" tag v0.0.3
  git -C "${SANDBOX}" tag "${NESTED}/v0.0.3"
  commit "release 0.0.4"
  git -C "${SANDBOX}" tag v0.0.4
  run "${SANDBOX}/scripts/release-pages.sh" root 0.0.4
  [ "${status}" -eq 0 ]
  [[ "${output}" == *$'-previous\nv0.0.3\n'* ]]
  [[ "${output}" != *"-target"* ]]
}

@test "a tag that does not exist yet is placed at HEAD" {
  git -C "${SANDBOX}" tag "${NESTED}/v0.0.4"
  commit "work towards 0.0.5"
  run "${SANDBOX}/scripts/release-pages.sh" processor 0.0.5
  [ "${status}" -eq 0 ]
  [[ "${output}" == *$'-previous\n'"${NESTED}/v0.0.4"* ]]
  [[ "${output}" == *$'-target\n'"$(git -C "${SANDBOX}" rev-parse HEAD)"* ]]
  [[ "${output}" == *"docs/releases/agentgateway-extmcp/v0.0.5.md"* ]]
}

@test "a malformed version or kind is a usage error" {
  run "${SANDBOX}/scripts/release-pages.sh" root 0.0
  [ "${status}" -eq 2 ]
  run "${SANDBOX}/scripts/release-pages.sh" gateway 0.0.4
  [ "${status}" -eq 2 ]
}
