#!/usr/bin/env bash
set -euo pipefail

temporary_directory=''
script_directory=''

main() {
  script_directory=$(cd "$(dirname "$0")" && pwd)
  temporary_directory=$(mktemp -d /tmp/wade-release-tests.XXXXXX)
  trap 'rm -rf "$temporary_directory"' EXIT
  mkdir "$temporary_directory/mock-bin" "$temporary_directory/package"

  cat > "$temporary_directory/mock-bin/git" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  'rev-parse HEAD') printf '%s\n' "${RELEASE_TEST_HEAD:-tagged-commit}" ;;
  rev-parse\ refs/tags/*) printf 'tagged-commit\n' ;;
  'merge-base --is-ancestor tagged-commit refs/remotes/origin/main') exit "${RELEASE_TEST_ANCESTOR_STATUS:-0}" ;;
  *) exit 99 ;;
esac
MOCK
  cat > "$temporary_directory/mock-bin/gh" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
if [[ ${RELEASE_TEST_API_FAILURE:-0} == 1 ]]; then
  exit 1
fi
printf '%s\n' "$RELEASE_TEST_PAGES"
MOCK
  chmod +x "$temporary_directory/mock-bin/git" "$temporary_directory/mock-bin/gh"
  export RELEASE_TEST_PAGES='[[]]'

  assert_fails 'Usage:' check_release
  local invalid_tag
  for invalid_tag in v1 v01.2.3 0.1.0 v0.1.0-beta.0 v0.1.0-beta.01 v0.1.0-rc.1 v0.1.0+build 'v0.1.0;false'; do
    assert_fails 'Expected vX.Y.Z' check_release "$invalid_tag"
  done
  check_release v0.1.0-beta.1
  check_release v0.1.0-beta.12
  check_release v0.1.0
  RELEASE_TEST_HEAD=another-commit assert_fails 'no longer matches' check_release v0.1.0-beta.1
  RELEASE_TEST_ANCESTOR_STATUS=1 assert_fails 'already on origin/main' check_release v0.1.0-beta.1
  RELEASE_TEST_API_FAILURE=1 assert_fails 'release lookup failed' check_release v0.1.0-beta.1
  RELEASE_TEST_PAGES=invalid-json assert_fails 'release lookup failed' check_release v0.1.0-beta.1
  RELEASE_TEST_PAGES='[[{"tag_name":"v0.1.0-beta.1","draft":true}]]' check_release v0.1.0-beta.1
  RELEASE_TEST_PAGES='[[{"tag_name":"v0.1.0-beta.2","draft":false}]]' check_release v0.1.0-beta.1
  RELEASE_TEST_PAGES='[[],[{"tag_name":"v0.1.0-beta.1","draft":false}]]' \
    assert_fails 'already published' check_release v0.1.0-beta.1

  local operating_system architecture
  case "$(uname -s)" in
    Darwin) operating_system=darwin ;;
    Linux) operating_system=linux ;;
    *) printf 'Unsupported smoke-test platform.\n' >&2; exit 1 ;;
  esac
  case "$(uname -m)" in
    arm64|aarch64) architecture=arm64 ;;
    x86_64) architecture=amd64 ;;
    *) printf 'Unsupported smoke-test architecture.\n' >&2; exit 1 ;;
  esac
  local version=v0.1.0-beta.1
  local archive_name="wade_${version#v}_${operating_system}_${architecture}.tar.gz"
  local archive_path="$temporary_directory/$archive_name"
  local checksum_path="$temporary_directory/checksums.txt"

  cat > "$temporary_directory/package/wade" <<'BINARY'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  help) printf 'wade v0.0.0\n' ;;
  stop) exit 0 ;;
  *) printf 'Unexpected binary execution.\n' >&2; exit 1 ;;
esac
BINARY
  chmod +x "$temporary_directory/package/wade"
  printf 'Licence\n' > "$temporary_directory/package/LICENSE"
  printf 'Readme\n' > "$temporary_directory/package/README.md"
  package_fixture "$archive_path" "$checksum_path" LICENSE README.md wade

  assert_fails 'Usage:' smoke_release
  assert_fails 'does not match the native platform' smoke_release "$temporary_directory/wrong.tar.gz" "$checksum_path" "$version"
  assert_fails 'leading v' smoke_release "$archive_path" "$checksum_path" '0.1.0-beta.1'
  assert_fails 'wrong version' smoke_release "$archive_path" "$checksum_path" "$version"

  cp "$checksum_path" "$temporary_directory/original-checksums.txt"
  cat "$temporary_directory/original-checksums.txt" >> "$checksum_path"
  assert_fails 'exactly one checksum' smoke_release "$archive_path" "$checksum_path" "$version"
  : > "$checksum_path"
  assert_fails 'exactly one checksum' smoke_release "$archive_path" "$checksum_path" "$version"
  cp "$temporary_directory/original-checksums.txt" "$checksum_path"
  printf 'corruption\n' >> "$archive_path"
  assert_fails 'FAILED' smoke_release "$archive_path" "$checksum_path" "$version"

  printf 'Unexpected file\n' > "$temporary_directory/package/extra.txt"
  package_fixture "$archive_path" "$checksum_path" LICENSE README.md wade extra.txt
  assert_fails 'must contain only' smoke_release "$archive_path" "$checksum_path" "$version"
  package_fixture "$archive_path" "$checksum_path" README.md wade
  assert_fails 'must contain only' smoke_release "$archive_path" "$checksum_path" "$version"
  chmod -x "$temporary_directory/package/wade"
  package_fixture "$archive_path" "$checksum_path" LICENSE README.md wade
  assert_fails 'not executable' smoke_release "$archive_path" "$checksum_path" "$version"
  rm "$temporary_directory/package/wade"
  ln -s README.md "$temporary_directory/package/wade"
  package_fixture "$archive_path" "$checksum_path" LICENSE README.md wade
  assert_fails 'regular file: wade' smoke_release "$archive_path" "$checksum_path" "$version"

  printf 'Release validation script tests passed.\n'
}

assert_fails() {
  local expected_message=$1
  shift
  if "$@" > "$temporary_directory/failure.log" 2>&1; then
    printf 'Expected failure: %s\n' "$*" >&2
    exit 1
  fi
  if ! grep -F "$expected_message" "$temporary_directory/failure.log" > /dev/null; then
    cat "$temporary_directory/failure.log" >&2
    printf 'Expected failure message: %s\n' "$expected_message" >&2
    exit 1
  fi
}

check_release() {
  env PATH="$temporary_directory/mock-bin:$PATH" GITHUB_REPOSITORY=example/wade \
    bash "$script_directory/check-release.sh" "$@"
}

smoke_release() {
  bash "$script_directory/smoke-release.sh" "$@"
}

package_fixture() {
  local archive_path=$1 checksum_path=$2
  shift 2
  COPYFILE_DISABLE=1 tar -czf "$archive_path" -C "$temporary_directory/package" "$@"
  (cd "$(dirname "$archive_path")" && shasum -a 256 "$(basename "$archive_path")") > "$checksum_path"
}

main "$@"
