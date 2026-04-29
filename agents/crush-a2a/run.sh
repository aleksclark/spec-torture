#!/usr/bin/env bash
#
# Run the A2A v1.0 spec-torture suite against crush-a2a (Crush with A2A plugin).
#
# Usage: run.sh <spec-yaml> <reports-dir> <spec-torture-binary>
#
set -euo pipefail

SPEC="${1:?Usage: run.sh <spec> <reports-dir> <binary>}"
REPORTS="${2:?}"
BINARY="${3:?}"
RUNTIME="crush-a2a"
PORT=8203
RPC_PATH="/"
DIR="$(cd "$(dirname "$0")" && pwd)"
PIDS=()

# Path to crush-a2a binary — override via CRUSH_A2A_BIN env var
CRUSH_BIN="${CRUSH_A2A_BIN:-/tmp/crush-a2a/crush}"

cleanup() {
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    done
}
trap cleanup EXIT

# ── 1. Verify binary exists ──────────────────────────────────────────
if [ ! -x "$CRUSH_BIN" ]; then
    echo "✗ crush-a2a binary not found at $CRUSH_BIN" >&2
    echo "  Set CRUSH_A2A_BIN to the path of the crush binary with A2A plugin" >&2
    exit 1
fi

echo "▸ crush-a2a binary: $CRUSH_BIN"
echo "  $($CRUSH_BIN --version 2>&1 || echo 'unknown version')"

# ── 2. Start crush in A2A serve mode ─────────────────────────────────
# Use an isolated home directory to avoid interfering with the user's config.
export HOME="$DIR/.crush-home"
mkdir -p "$HOME"

# Write a minimal crush.json that configures the A2A server port
mkdir -p "$HOME/.config/crush"
cat > "$HOME/.config/crush/crush.json" <<CONF
{
  "options": {
    "plugins": {
      "a2a-server": {
        "port": $PORT
      }
    },
    "disabled_plugins": ["otlp", "agent-status", "periodic-prompts", "tempotown"]
  }
}
CONF

echo "▸ Starting crush-a2a on port $PORT..."
"$CRUSH_BIN" serve >/dev/null 2>&1 &
PIDS+=($!)

echo "▸ Waiting for crush-a2a on port $PORT..."
for i in $(seq 1 30); do
    python3 -c "
import urllib.request
try:
    urllib.request.urlopen('http://localhost:$PORT/.well-known/agent-card.json', timeout=2)
    exit(0)
except:
    exit(1)
" 2>/dev/null && break
    sleep 1
done

if ! python3 -c "
import urllib.request
try:
    urllib.request.urlopen('http://localhost:$PORT/.well-known/agent-card.json', timeout=2)
    exit(0)
except:
    exit(1)
" 2>/dev/null; then
    echo "✗ crush-a2a failed to start on port $PORT" >&2
    echo "  Logs:" >&2
    cat "$HOME/.config/crush/logs/"*.log 2>/dev/null | tail -20 >&2 || true
    exit 1
fi
echo "✓ crush-a2a: http://localhost:$PORT (JSON-RPC at $RPC_PATH)"

# ── 3. Run suite ─────────────────────────────────────────────────────
echo "▸ Running A2A v1.0 suite..."
mkdir -p "$REPORTS"

"$BINARY" run "$SPEC" \
    --runtime "$RUNTIME" \
    --url "http://localhost:$PORT" \
    --rpc-path "$RPC_PATH" \
    > "$REPORTS/$RUNTIME.md" \
    2>"$REPORTS/$RUNTIME.log"

"$BINARY" run "$SPEC" \
    --runtime "$RUNTIME" \
    --url "http://localhost:$PORT" \
    --rpc-path "$RPC_PATH" \
    --format json \
    > "$REPORTS/$RUNTIME.json" \
    2>"$REPORTS/$RUNTIME-json.log"

# ── 4. Report ────────────────────────────────────────────────────────
SCORE=$(grep 'Compliance' "$REPORTS/$RUNTIME.md" | grep -o '[0-9]*\.[0-9]*%' || echo "?")
echo "✓ $RUNTIME: $SCORE"
echo "  Reports: $REPORTS/$RUNTIME.md, $REPORTS/$RUNTIME.json"
