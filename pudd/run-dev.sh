#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
  echo "Run this script without sudo; it will build first, then elevate only to start pudd." >&2
  exit 1
fi

GO_BIN="${GO_BIN:-}"
if [[ -z "$GO_BIN" ]]; then
  GO_BIN="$(command -v go || true)"
fi
if [[ -z "$GO_BIN" && -x /usr/local/go/bin/go ]]; then
  GO_BIN=/usr/local/go/bin/go
fi
if [[ -z "$GO_BIN" ]]; then
  echo "go not found in PATH. Set GO_BIN=/full/path/to/go if needed." >&2
  exit 1
fi

mkdir -p "$SCRIPT_DIR/bin" "$SCRIPT_DIR/etc" "$SCRIPT_DIR/tmp/mnt/_probe" "$SCRIPT_DIR/tmp/staging"

BUILD_OUT="$SCRIPT_DIR/bin/pudd-dev"
"$GO_BIN" build -o "$BUILD_OUT" ./cmd/pudd

exec sudo "$BUILD_OUT" \
  -bucket pudd \
  -prefix devices/dev-test-001 \
  -creds "$SCRIPT_DIR/etc/pudd-dev-sa.json" \
  -db "$SCRIPT_DIR/pudd.db" \
  -mount-root "$SCRIPT_DIR/tmp/mnt" \
  -probe-root "$SCRIPT_DIR/tmp/mnt/_probe" \
  -stage-root "$SCRIPT_DIR/tmp/staging" \
  "$@"
