#!/usr/bin/env bash
#
# Run the A2A v1.0 spec-torture suite against the official a2a-go helloworld agent.
#
# Usage: run.sh <spec-yaml> <reports-dir> <spec-torture-binary>
#
set -euo pipefail

SPEC="${1:?Usage: run.sh <spec> <reports-dir> <binary>}"
REPORTS="${2:?}"
BINARY="${3:?}"
RUNTIME="a2a-go-helloworld"
PORT=8201
RPC_PATH="/invoke"
DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD="$DIR/.build"
PIDS=()

cleanup() {
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    done
}
trap cleanup EXIT

# ── 1. Clone / update ────────────────────────────────────────────────
mkdir -p "$BUILD"
if [ ! -d "$BUILD/a2a-go" ]; then
    echo "▸ Cloning a2a-go..."
    git clone --depth 1 https://github.com/a2aproject/a2a-go.git "$BUILD/a2a-go"
else
    echo "▸ Updating a2a-go..."
    git -C "$BUILD/a2a-go" pull --ff-only 2>/dev/null || true
fi

# ── 2. Build and start ───────────────────────────────────────────────
echo "▸ Building a2a-go helloworld..."
(cd "$BUILD/a2a-go" && go build -o helloworld-jsonrpc ./examples/helloworld/server/jsonrpc)

"$BUILD/a2a-go/helloworld-jsonrpc" --port "$PORT" >/dev/null 2>&1 &
PIDS+=($!)

echo "▸ Waiting for a2a-go on port $PORT..."
for i in $(seq 1 30); do
    curl -s "http://localhost:$PORT/.well-known/agent-card.json" >/dev/null 2>&1 && break
    sleep 1
done
if ! curl -s "http://localhost:$PORT/.well-known/agent-card.json" >/dev/null 2>&1; then
    echo "✗ a2a-go helloworld failed to start on port $PORT" >&2
    exit 1
fi
echo "✓ a2a-go: http://localhost:$PORT (JSON-RPC at $RPC_PATH)"

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
