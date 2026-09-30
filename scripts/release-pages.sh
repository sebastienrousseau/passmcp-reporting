#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# Compose, and with --publish put on GitHub, one release page of a
# passmcp-reporting release in the family layout. There are two per
# release: `root` for the module's tag vX.Y.Z, titled passmcp-reporting,
# and `processor` for the agentgateway processor's nested tag
# integrations/agentgateway-extmcp/vX.Y.Z, titled after the processor and
# never marked latest.
#
# scripts/releasepage writes each page. This supplies what differs between
# the two: the tag, the title, the highlights file, the tag the change list
# starts from, and, because neither release attaches files, the digest the
# registry serves for the processor's image at that version.
#
#   scripts/release-pages.sh [--publish] root|processor X.Y.Z
#
# Without --publish the page is printed and nothing is written; a tag that
# does not exist yet is placed at HEAD for GitHub's generated notes, and an
# image that is not published yet is named as such.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

readonly IMAGE_PATH=sebastienrousseau/passmcp-agentgateway-extmcp
readonly NESTED=integrations/agentgateway-extmcp

# digest prints the multi-arch digest ghcr.io serves for the version, or
# nothing. Fetched to a file and parsed from it, as passmcp's release does.
digest() {
  local token
  curl -fsSL "https://ghcr.io/token?scope=repository:${IMAGE_PATH}:pull" -o "${work}/token.json" || return 0
  token=$(jq -r .token "${work}/token.json")
  curl -fsSI -H "Authorization: Bearer ${token}" \
    -H "Accept: application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json" \
    "https://ghcr.io/v2/${IMAGE_PATH}/manifests/$1" 2>/dev/null \
    | tr -d '\r' | awk 'tolower($1) == "docker-content-digest:" { print $2 }'
}

# since prints the tag of the same series the change list starts from: the
# nearest one before the tag, or before HEAD when the tag does not exist.
since() {
  local tag=$1 match=$2 rev=HEAD
  git rev-parse -q --verify "refs/tags/${tag}" >/dev/null && rev="${tag}^"
  git describe --tags --abbrev=0 --match "${match}" "${rev}" 2>/dev/null || true
}

# image_note is the checksums section's account of what is published
# instead of files. A published page must name the real digest.
image_note() {
  local kind=$1 ver=$2 publish=$3 sha
  sha=$(digest "${ver}")
  if [ -z "${sha}" ]; then
    if [ "${publish}" = yes ]; then
      echo "release-pages: ghcr.io/${IMAGE_PATH}:${ver} is not published; re-run once the v${ver} release has pushed it" >&2
      return 1
    fi
    sha="sha256:<not published yet>"
  fi
  local lead="The agentgateway processor is published as a container image instead"
  [ "${kind}" = processor ] && lead="The processor is published as a container image"
  echo "${lead}: \`ghcr.io/${IMAGE_PATH}:${ver}\` is \`${sha}\`, with SLSA build provenance attached to the digest."
}

# page_args sets tag, match and args for one kind of page.
page_args() {
  local kind=$1 ver=$2
  case "${kind}" in
    root)
      tag="v${ver}" match='v[0-9]*'
      args=(-name passmcp-reporting -tag "${tag}") ;;
    processor)
      tag="${NESTED}/v${ver}" match="${NESTED}/v[0-9]*"
      args=(-name agentgateway-extmcp -tag "${tag}" -latest=false
        -highlights "docs/releases/agentgateway-extmcp/v${ver}.md") ;;
    *) return 1 ;;
  esac
}

main() {
  local publish=no
  if [ "${1:-}" = --publish ]; then publish=yes; shift; fi
  local kind=${1:-} ver=${2:-} tag match args prev note
  if [[ ! "${ver}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || ! page_args "${kind}" "${ver}"; then
    echo "usage: $0 [--publish] root|processor X.Y.Z" >&2
    return 2
  fi
  prev=$(since "${tag}" "${match}")
  [ -z "${prev}" ] || args+=(-previous "${prev}")
  git rev-parse -q --verify "refs/tags/${tag}" >/dev/null || args+=(-target "$(git rev-parse HEAD)")
  [ "${publish}" = no ] || args+=(-publish)
  note=$(image_note "${kind}" "${ver}" "${publish}")
  go run ./scripts/releasepage "${args[@]}" -note "${note}"
}

work=$(mktemp -d)
trap 'rm -rf "${work}"' EXIT
main "$@"
