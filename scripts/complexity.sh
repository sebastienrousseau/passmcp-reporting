#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# The complexity ceilings, against a committed baseline of the functions
# already over them. The ceilings are the portfolio's (cyclomatic 10,
# cognitive 15, 60 lines per function, set in .golangci.yml) and 500 lines
# per hand-written Go file (set here). The baseline may only shrink:
#
#   - a function or file over a ceiling that is not in the baseline fails;
#   - a baseline entry whose measure grew fails;
#   - a baseline entry now within its ceiling, or smaller than recorded,
#     fails until the baseline records it, so an improvement is kept.
#
#   scripts/complexity.sh            check against the baseline
#   scripts/complexity.sh --update   record improvements; refuses to
#                                    record a new or a worse offender
#
# The measures come from golangci-lint (gocyclo, gocognit, funlen) run in
# the root module and in each nested module under integrations/, with the
# root's configuration and its exclusions, so tests are not measured.
# GOLANGCI_LINT replaces the golangci-lint command, for the tests.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
root=$(pwd)
readonly baseline=scripts/complexity-baseline.txt
readonly max_file_lines=500
read -r -a lint <<<"${GOLANGCI_LINT:-golangci-lint}"

# measure prints one "<measure>\t<path>\t<name>\t<value>" line per
# function or file over a ceiling, sorted.
measure() {
  local dir rc out
  local modules=(.)
  for dir in integrations/*/; do
    [ -f "${dir}go.mod" ] && modules+=("${dir%/}")
  done
  for dir in "${modules[@]}"; do
    rc=0
    out=$(cd "${dir}" && "${lint[@]}" run --config "${root}/.golangci.yml" \
      --enable-only gocyclo,gocognit,funlen \
      --max-issues-per-linter 0 --max-same-issues 0 \
      --output.json.path stdout --show-stats=false ./...) || rc=$?
    # 1 is "issues found", which is what is being measured.
    if [ "${rc}" -gt 1 ]; then
      echo "complexity: golangci-lint failed in ${dir} (exit ${rc})" >&2
      printf '%s\n' "${out}" >&2
      exit 1
    fi
    # A message neither pattern reads comes out as "unparsed" and fails
    # below, rather than being dropped: a linter changing its wording must
    # not make every offender disappear. The function name in funlen's
    # message is quoted; "." matches the quotes.
    jq -r '.Issues[]? | .FromLinter as $l | .Pos.Filename as $f | .Text as $t |
      ([ $t | capture("complexity (?<v>[0-9]+) of func `(?<n>[^`]+)`"),
              capture("Function .(?<n>[^ ]+). (?<k>is too long|has too many statements) \\((?<v>[0-9]+) >") ]
        | first // {n: $t, v: "unparsed"})
      | [ (if $l == "funlen" then (if .k == "is too long" then "funlen-lines" else "funlen-statements" end) else $l end),
          $f, .n, .v ] | @tsv' <<<"${out}"
  done
  # Hand-written Go files over the per-file ceiling; generated code is the
  # generator's, not a file anyone reads to review.
  git ls-files -co --exclude-standard -- '*.go' | while read -r f; do
    [ -f "${f}" ] || continue
    grep -q '^// Code generated .* DO NOT EDIT\.$' "${f}" && continue
    n=$(wc -l <"${f}" | tr -d ' ')
    [ "${n}" -gt "${max_file_lines}" ] && printf 'file-lines\t%s\t-\t%s\n' "${f}" "${n}"
    true
  done
}

current=$(measure | LC_ALL=C sort)
unparsed=$(awk -F'\t' '$4 == "unparsed"' <<<"${current}")
if [ -n "${unparsed}" ]; then
  echo "complexity: a linter message this script cannot read:" >&2
  printf '%s\n' "${unparsed}" >&2
  exit 1
fi
recorded=$(grep -v '^#' "${baseline}" 2>/dev/null | grep -v '^$' | LC_ALL=C sort || true)

# compare prints one line per difference, prefixed with its kind: "new",
# "worse", "fixed" or "better".
compare() {
  awk -F'\t' '
    FILENAME == ARGV[1] { base[$1 FS $2 FS $3] = $4; next }
    { key = $1 FS $2 FS $3; now[key] = $4
      if (!(key in base)) print "new\t" $0
      else if ($4 + 0 > base[key] + 0) print "worse\t" $0 "\t" base[key]
      else if ($4 + 0 < base[key] + 0) print "better\t" $0 "\t" base[key] }
    END { for (key in base) if (!(key in now)) print "fixed\t" key "\t" base[key] }
  ' <(printf '%s\n' "${recorded}" | sed '/^$/d') <(printf '%s\n' "${current}" | sed '/^$/d') | LC_ALL=C sort
}

diffs=$(compare)
regressions=$(grep -E '^(new|worse)' <<<"${diffs}" || true)

if [ "${1:-}" = "--update" ]; then
  if [ -n "${regressions}" ]; then
    echo "complexity: refusing to record a new or worse offender; fix it instead:" >&2
    printf '%s\n' "${regressions}" >&2
    exit 1
  fi
  {
    # REUSE-IgnoreStart
    echo "# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>"
    echo "# SPDX-License-Identifier: Apache-2.0"
    # REUSE-IgnoreEnd
    echo "#"
    echo "# The functions and files over the complexity ceilings, and by how much."
    echo "# Written by scripts/complexity.sh --update; it may only shrink."
    echo "# <measure>	<path>	<function>	<value>"
    [ -z "${current}" ] || printf '%s\n' "${current}"
  } >"${baseline}"
  echo "complexity: recorded $(printf '%s' "${current}" | grep -c . || true) offenders in ${baseline}"
  exit 0
fi

if [ -z "${diffs}" ]; then
  echo "complexity: $(printf '%s' "${current}" | grep -c . || true) offenders, none new, none worse"
  exit 0
fi
while IFS=$'\t' read -r kind measure path name value was; do
  case "${kind}" in
    new) echo "complexity: new offender: ${path} ${name}: ${measure} ${value}" >&2 ;;
    worse) echo "complexity: worse: ${path} ${name}: ${measure} ${was} -> ${value}" >&2 ;;
    better) echo "complexity: improved but not recorded: ${path} ${name}: ${measure} ${was} -> ${value}" >&2 ;;
    fixed) echo "complexity: now within the ceiling but still in the baseline: ${path} ${name} (${measure} was ${value})" >&2 ;;
  esac
done <<<"${diffs}"
[ -z "${regressions}" ] || echo "complexity: bring the function under the ceiling; the baseline never absorbs a regression" >&2
[ -n "${regressions}" ] || echo "complexity: run scripts/complexity.sh --update to record the improvement" >&2
exit 1
