#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# Fail unless every version-bearing place agrees on the version being
# released: the CHANGELOG heading, the release notes, CITATION.cff, the
# README's install snippet and family sentence, the agentgateway
# processor's README and image, and the processor's go.mod requiring a
# released root version no newer than this one.
#
#   scripts/verify-release-versions.sh v0.0.1
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
tag="${1:-${GITHUB_REF_NAME:-}}"
[ -n "$tag" ] || { echo "usage: $0 vX.Y.Z" >&2; exit 2; }
ver="${tag#v}"
grep -Eq "^## \[$ver\]" CHANGELOG.md || { echo "CHANGELOG.md has no '## [$ver]' heading" >&2; exit 1; }
if grep -Eo 'passmcp-reporting@v[0-9]+\.[0-9]+\.[0-9]+' README.md | grep -v "passmcp-reporting@v$ver"; then
  echo "README.md pins a version other than $ver" >&2; exit 1
fi
proc=integrations/agentgateway-extmcp/README.md
if grep -Eo 'agentgateway-extmcp@v[0-9]+\.[0-9]+\.[0-9]+' "$proc" | grep -v "agentgateway-extmcp@v$ver"; then
  echo "$proc pins a version other than $ver" >&2; exit 1
fi
grep -q "agentgateway-extmcp@v$ver" "$proc" || { echo "$proc has no install snippet for v$ver" >&2; exit 1; }
if grep -Eo 'agentgateway-extmcp:[0-9]+\.[0-9]+\.[0-9]+' "$proc" | grep -v "agentgateway-extmcp:$ver"; then
  echo "$proc names an image version other than $ver" >&2; exit 1
fi
[ -f "docs/releases/v$ver.md" ] || { echo "docs/releases/v$ver.md is missing" >&2; exit 1; }
grep -q '^## Highlights' "docs/releases/v$ver.md" || { echo "docs/releases/v$ver.md has no Highlights section" >&2; exit 1; }
grep -Eq "^version: \"?$ver\"?$" CITATION.cff || { echo "CITATION.cff does not say version $ver" >&2; exit 1; }
grep -q "Every component is released at \*\*$ver\*\*" README.md || { echo "README.md's ecosystem section does not say $ver" >&2; exit 1; }
# The processor requires a released root version, never a pseudo-version,
# and never one after the release being cut. It cannot require this
# release itself: the tag does not exist until this commit is tagged, so
# a verifier change the processor needs ships in two steps (ADR 0002).
nested=integrations/agentgateway-extmcp/go.mod
req=$(sed -nE 's/^[[:space:]]*satellion\.com\/passmcp-reporting (v[^[:space:]]+).*$/\1/p' "$nested")
[[ "$req" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
  echo "$nested requires satellion.com/passmcp-reporting ${req:-nothing}, not a released version" >&2; exit 1
}
[ "$(printf '%s\n' "${req#v}" "$ver" | sort -t. -k1,1n -k2,2n -k3,3n | tail -1)" = "$ver" ] || {
  echo "$nested requires satellion.com/passmcp-reporting $req, newer than $ver" >&2; exit 1
}
echo "release versions agree on $ver"
