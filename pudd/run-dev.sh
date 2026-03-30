#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

mkdir -p ./etc ./tmp/mnt/_probe ./tmp/staging

/usr/local/go/bin/go run ./cmd/pudd \
  -bucket pudd \
  -prefix devices/dev-test-001 \
  -creds ./etc/pudd-dev-sa.json \
  -db ./pudd.db \
  -mount-root ./tmp/mnt \
  -probe-root ./tmp/mnt/_probe \
  -stage-root ./tmp/staging \
  "$@"
