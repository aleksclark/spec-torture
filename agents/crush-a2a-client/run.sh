#!/usr/bin/env bash
#
# Run the A2A v1.0 client conformance suite against crush-a2a's A2A client tools.
#
# The spec-torture run-client command starts its own mock A2A server, then
# uses the crush binary to exercise a2a_send_message, a2a_get_task, etc.
# against the mock, and evaluates protocol conformance.
#
# Usage: run.sh <reports-dir> <spec-torture-binary>
#
set -euo pipefail

REPORTS="${1:?Usage: run.sh <reports-dir> <spec-torture-binary>}"
BINARY="${2:?}"
RUNTIME="crush-a2a-client"
DIR="$(cd "$(dirname "$0")" && pwd)"
CRUSH_BIN="${CRUSH_A2A_BIN:-/tmp/crush-a2a/crush}"
CRUSH_HOME="$DIR/.crush-home"

if [ ! -x "$CRUSH_BIN" ]; then
    echo "✗ crush binary not found at $CRUSH_BIN" >&2
    echo "  Set CRUSH_A2A_BIN to override" >&2
    exit 1
fi

echo "▸ crush binary: $CRUSH_BIN"
echo "  $($CRUSH_BIN --version 2>&1 || echo 'unknown version')"

mkdir -p "$CRUSH_HOME/.config/crush" "$REPORTS"

echo "▸ Running A2A v1.0 client suite..."

"$BINARY" run-client \
    --runtime "$RUNTIME" \
    --crush-bin "$CRUSH_BIN" \
    --crush-home "$CRUSH_HOME" \
    > "$REPORTS/$RUNTIME.md" \
    2>"$REPORTS/$RUNTIME.log"

"$BINARY" run-client \
    --runtime "$RUNTIME" \
    --crush-bin "$CRUSH_BIN" \
    --crush-home "$CRUSH_HOME" \
    --format json \
    > "$REPORTS/$RUNTIME.json" \
    2>"$REPORTS/$RUNTIME-json.log"

SCORE=$(grep 'Compliance' "$REPORTS/$RUNTIME.md" | grep -o '[0-9]*\.[0-9]*%' || echo "?")
echo "✓ $RUNTIME: $SCORE"
echo "  Reports: $REPORTS/$RUNTIME.md, $REPORTS/$RUNTIME.json"
