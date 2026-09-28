#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# Fail unless CHANGELOG.md has a heading for the version being released and
# no install snippet pins a stale version: the root module's in README.md,
# and the agentgateway processor's in its own README.
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
echo "release versions agree on $ver"
