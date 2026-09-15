#!/usr/bin/env bash
set -euo pipefail

temporary_directory=''
binary=''
daemon_pid=''
foreground_pid=''

main() {
  if [[ $# != 3 ]]; then
    fail "Usage: $0 <archive.tar.gz> <checksums.txt> <expected-version>"
  fi

  local archive_path checksum_path expected_version=$3 operating_system architecture
  archive_path="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
  checksum_path="$(cd "$(dirname "$2")" && pwd)/$(basename "$2")"
  case "$(uname -s)" in
    Darwin) operating_system=darwin ;;
    Linux) operating_system=linux ;;
    *) fail 'Smoke tests require macOS or Linux.' ;;
  esac
  case "$(uname -m)" in
    arm64|aarch64) architecture=arm64 ;;
    x86_64) architecture=amd64 ;;
    *) fail 'Smoke tests require ARM64 or AMD64.' ;;
  esac

  local version_pattern='^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'
  [[ $expected_version =~ $version_pattern ]] || fail 'Expected a version with a leading v.'
  local archive_name="wade_${expected_version#v}_${operating_system}_${architecture}.tar.gz"
  [[ $(basename "$archive_path") == "$archive_name" ]] || fail 'Archive does not match the native platform and expected version.'

  # Keep control socket paths below macOS's Unix-domain socket path limit.
  temporary_directory=$(mktemp -d /tmp/wade-smoke.XXXXXX)
  trap cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM

  awk -v archive="$archive_name" '$2 == archive { print }' "$checksum_path" > "$temporary_directory/checksum.txt"
  [[ $(wc -l < "$temporary_directory/checksum.txt" | tr -d ' ') == 1 ]] || fail 'Expected exactly one checksum for the archive.'
  (cd "$(dirname "$archive_path")" && shasum -a 256 --check "$temporary_directory/checksum.txt")

  local entries
  entries=$(tar -tzf "$archive_path" | LC_ALL=C sort)
  [[ $entries == $'LICENSE\nREADME.md\nwade' ]] || fail 'Archive must contain only LICENSE, README.md and wade.'
  mkdir "$temporary_directory/extracted"
  tar -xzf "$archive_path" -C "$temporary_directory/extracted"
  local filename
  for filename in LICENSE README.md wade; do
    [[ -f "$temporary_directory/extracted/$filename" && ! -L "$temporary_directory/extracted/$filename" ]] || fail "Expected a regular file: $filename"
  done
  [[ -x "$temporary_directory/extracted/wade" ]] || fail 'The packaged binary is not executable.'

  export HOME="$temporary_directory/home"
  export XDG_CONFIG_HOME="$HOME/.config"
  export XDG_STATE_HOME="$temporary_directory/state"
  export XDG_CACHE_HOME="$temporary_directory/cache"
  export SHELL=/bin/sh
  export WADE_DEV=0
  unset WADE_INTERNAL_SERVER_READY_FD
  mkdir -p "$HOME/.config/wade" "$XDG_STATE_HOME" "$XDG_CACHE_HOME"
  printf '{"workspaceDirectories":[],"shell":"/bin/sh"}\n' > "$HOME/.config/wade/config.json"
  cd "$temporary_directory"
  binary="$temporary_directory/extracted/wade"

  "$binary" help > "$temporary_directory/help.txt"
  local first_line
  IFS= read -r first_line < "$temporary_directory/help.txt"
  [[ $first_line == "wade $expected_version" ]] || fail 'Packaged binary reports the wrong version.'

  local port
  port=$(python3 -c 'import socket; s = socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')
  export WADE_ADDR="127.0.0.1:$port"
  local server_url="http://$WADE_ADDR" socket_path="$XDG_STATE_HOME/wade/server.sock"

  "$binary" start > "$temporary_directory/start.txt"
  daemon_pid=$(awk '/^PID: / { print $2 }' "$temporary_directory/start.txt")
  [[ $daemon_pid =~ ^[0-9]+$ && $daemon_pid -gt 1 ]] || fail 'Daemon startup did not report a valid PID.'
  "$binary" status > "$temporary_directory/status.txt"
  grep -Fx "Version: $expected_version" "$temporary_directory/status.txt"
  grep -Fx "PID: $daemon_pid" "$temporary_directory/status.txt"
  grep -Fx "Address: $WADE_ADDR" "$temporary_directory/status.txt"
  [[ -S $socket_path ]] || fail 'Managed daemon did not create a control socket.'

  "$binary" start > "$temporary_directory/restart.txt"
  grep -Fx 'WADE is already running' "$temporary_directory/restart.txt"
  grep -Fx "PID: $daemon_pid" "$temporary_directory/restart.txt"

  curl --noproxy '*' --fail --silent --show-error --max-time 10 "$server_url/" > "$temporary_directory/index.html"
  grep -F 'id="app"' "$temporary_directory/index.html"
  grep -F 'src="/static/app.js"' "$temporary_directory/index.html"
  grep -F 'href="/static/app.css"' "$temporary_directory/index.html"
  curl --noproxy '*' --fail --silent --show-error --max-time 10 \
    --dump-header "$temporary_directory/js.headers" "$server_url/static/app.js" > "$temporary_directory/app.js"
  curl --noproxy '*' --fail --silent --show-error --max-time 10 \
    --dump-header "$temporary_directory/css.headers" "$server_url/static/app.css" > "$temporary_directory/app.css"
  grep -Ei '^Content-Type: (text|application)/javascript' "$temporary_directory/js.headers"
  grep -Ei '^Content-Type: text/css' "$temporary_directory/css.headers"
  [[ -s $temporary_directory/app.js && -s $temporary_directory/app.css ]] || fail 'Embedded frontend assets are empty.'

  "$binary" stop
  wait_for_exit "$daemon_pid" || fail 'Managed daemon did not exit.'
  daemon_pid=''
  assert_unmanaged "$socket_path"

  "$binary" start --foreground > "$temporary_directory/foreground.log" 2>&1 &
  foreground_pid=$!
  local attempt ready=false
  for ((attempt = 0; attempt < 100; attempt++)); do
    kill -0 "$foreground_pid" 2>/dev/null || fail 'Foreground server exited during startup.'
    if curl --noproxy '*' --fail --silent --max-time 1 "$server_url/" > /dev/null; then
      ready=true
      break
    fi
    sleep 0.1
  done
  [[ $ready == true ]] || fail 'Foreground server did not become ready.'
  assert_unmanaged "$socket_path"
  "$binary" stop
  kill -0 "$foreground_pid" || fail 'wade stop terminated the unmanaged foreground server.'
  curl --noproxy '*' --fail --silent --show-error --max-time 10 "$server_url/" > /dev/null
  kill -TERM "$foreground_pid"
  wait_for_exit "$foreground_pid" || fail 'Foreground server did not exit.'
  wait "$foreground_pid"
  foreground_pid=''
  assert_unmanaged "$socket_path"

  printf 'Release smoke test passed: %s (%s/%s)\n' "$expected_version" "$operating_system" "$architecture"
}

fail() {
  printf '%s\n' "$*" >&2
  exit 1
}

assert_unmanaged() {
  [[ ! -e $1 ]] || fail 'A stale control socket remains.'
  local status_output status_code
  if status_output=$("$binary" status 2>&1); then
    fail 'Expected no managed daemon.'
  else
    status_code=$?
  fi
  [[ $status_code == 1 && $status_output == 'WADE is not running' ]] || fail 'Unexpected stopped-daemon status.'
}

wait_for_exit() {
  local attempt
  for ((attempt = 0; attempt < 100; attempt++)); do
    if ! kill -0 "$1" 2>/dev/null; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

cleanup() {
  local exit_code=$? process_id log_path
  trap - EXIT
  set +e
  if [[ $exit_code != 0 ]]; then
    for log_path in "$temporary_directory/state/wade/server.log" "$temporary_directory/foreground.log"; do
      if [[ -f $log_path ]]; then
        printf '\n%s\n' "$log_path" >&2
        cat "$log_path" >&2
      fi
    done
  fi
  if [[ -n $binary ]]; then
    "$binary" stop >/dev/null 2>&1
  fi
  for process_id in "$daemon_pid" "$foreground_pid"; do
    if [[ $process_id =~ ^[0-9]+$ && $process_id -gt 1 ]] && kill -0 "$process_id" 2>/dev/null; then
      kill -TERM "$process_id" 2>/dev/null
      if ! wait_for_exit "$process_id"; then
        kill -KILL "$process_id" 2>/dev/null
      fi
    fi
  done
  if [[ -n $foreground_pid ]]; then
    wait "$foreground_pid" 2>/dev/null
  fi
  rm -rf "$temporary_directory"
  exit "$exit_code"
}

main "$@"
