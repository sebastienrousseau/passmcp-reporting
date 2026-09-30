#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# The root module's public API against the release before this commit:
# fail on an incompatible change, or when gorelease could not compare.
#
#   scripts/api-check.sh
#
# The base is the newest root release tag (bare vX.Y.Z) that HEAD contains
# and that is not on HEAD itself. On a release branch that is the last
# release; on main just after a release, HEAD carries the new tag, and
# comparing a tag with itself compares nothing, so the base is the release
# before it. The nested processor's tags are another module's and never a
# base here: when the base was once taken from the newest tag of any kind,
# it was the nested tag, gorelease could not use it, and the check
# reported "no incompatible change" having compared nothing. A gorelease
# that did not run is a failure, not a pass.
#
# gorelease also exits 1 when it merely cannot suggest the next version
# because the base is not the newest release the module proxy lists. That
# is always the case on main after a release, and on any branch while the
# proxy has not yet listed a tag that was just pushed. It says nothing
# about the API, so that one refusal, with no other diagnostic, passes.
# GORELEASE replaces the gorelease command, for the tests.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

read -r -a gorelease <<<"${GORELEASE:-go run golang.org/x/exp/cmd/gorelease@latest}"
readonly not_newest='^Can only suggest a release version when compared against the most recent version'

base=$(git tag --merged HEAD --no-contains HEAD --list 'v[0-9]*' --sort=-v:refname | head -1)
if [ -z "${base}" ]; then
  echo "api-check: no earlier release, nothing to compare"
  exit 0
fi

rc=0
out=$("${gorelease[@]}" -base="${base}" 2>&1) || rc=$?
printf '%s\n' "${out}"

if grep -qiE 'incompatible changes' <<<"${out}"; then
  echo "api-check: the public API changed incompatibly against ${base}" >&2
  exit 1
fi
if [ "${rc}" -ne 0 ]; then
  if grep -qE "${not_newest}" <<<"${out}" && ! grep -q '^# diagnostics' <<<"${out}"; then
    echo "api-check: no incompatible change against ${base} (gorelease suggests no version: ${base} is not the newest release the proxy lists)"
    exit 0
  fi
  echo "api-check: gorelease did not complete against ${base}" >&2
  exit 1
fi
echo "api-check: no incompatible change against ${base}"
