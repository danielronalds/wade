#!/usr/bin/env bash
set -euo pipefail

main() {
  if [[ $# != 1 ]]; then
    printf 'Usage: %s <version-tag>\n' "$0" >&2
    exit 1
  fi

  local release_tag=$1
  local version_pattern='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-beta\.[1-9][0-9]*)?$'
  if [[ ! $release_tag =~ $version_pattern ]]; then
    printf 'Expected vX.Y.Z or vX.Y.Z-beta.N (N starts at 1), got %s\n' "$release_tag" >&2
    exit 1
  fi

  local tagged_commit checked_out_commit
  tagged_commit=$(git rev-parse "refs/tags/${release_tag}^{commit}")
  checked_out_commit=$(git rev-parse HEAD)
  if [[ $tagged_commit != "$checked_out_commit" ]]; then
    printf 'The release tag no longer matches the checked-out commit.\n' >&2
    exit 1
  fi
  if ! git merge-base --is-ancestor "$tagged_commit" refs/remotes/origin/main; then
    printf 'Release tags must point to commits already on origin/main.\n' >&2
    exit 1
  fi

  : "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY must identify the release repository}"
  if ! gh api --paginate --slurp "repos/${GITHUB_REPOSITORY}/releases?per_page=100" |
    jq -e --arg tag "$release_tag" 'all(.[][]; .tag_name != $tag or .draft == true)' >/dev/null; then
    printf 'Cannot release: the tag is already published, or release lookup failed.\n' >&2
    exit 1
  fi
}

main "$@"
