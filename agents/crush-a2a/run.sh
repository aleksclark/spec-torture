#!/usr/bin/env bash
#
# Run the A2A v1.0 spec-torture suite against crush-a2a.
#
# Usage: run.sh <spec-yaml> <reports-dir> <spec-torture-binary>
#
set -euo pipefail

SPEC="${1:?Usage: run.sh <spec> <reports-dir> <binary>}"
REPORTS="${2:?}"
BINARY="${3:?}"
RUNTIME="crush-a2a"
PORT=8200
DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD="$DIR/.build"
PIDS=()

cleanup() {
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    done
    rm -f "/tmp/crush-a2a-torture-$$.sock"
}
trap cleanup EXIT

# ── 1. Start crush server ────────────────────────────────────────────
SOCK="/tmp/crush-$(id -u).sock"
if ! test -S "$SOCK"; then
    echo "▸ Starting crush server..."
    crush server >/dev/null 2>&1 &
    PIDS+=($!)
    for i in $(seq 1 30); do
        test -S "$SOCK" && break
        sleep 1
    done
    if ! test -S "$SOCK"; then
        echo "✗ crush server failed to start (no socket at $SOCK)" >&2
        exit 1
    fi
fi
echo "✓ crush server: $SOCK"

# ── 2. Build crush-a2a ───────────────────────────────────────────────
mkdir -p "$BUILD"
if [ ! -d "$BUILD/crush-a2a" ]; then
    echo "▸ Cloning crush-a2a..."
    git clone --depth 1 https://github.com/aleksclark/crush-a2a.git "$BUILD/crush-a2a"
else
    echo "▸ Updating crush-a2a..."
    git -C "$BUILD/crush-a2a" pull --ff-only 2>/dev/null || true
fi

echo "▸ Building crush-a2a..."
(cd "$BUILD/crush-a2a" && go build -o crush-a2a ./cmd/crush-a2a)

# ── 3. Start crush-a2a ───────────────────────────────────────────────
WORKSPACE="/tmp/crush-a2a-torture-$$"
mkdir -p "$WORKSPACE"
"$BUILD/crush-a2a/crush-a2a" \
    --crush-addr "unix://$SOCK" \
    --workspace-path "$WORKSPACE" \
    --port "$PORT" \
    >/dev/null 2>&1 &
PIDS+=($!)

echo "▸ Waiting for crush-a2a on port $PORT..."
for i in $(seq 1 30); do
    curl -s "http://localhost:$PORT/.well-known/agent-card.json" >/dev/null 2>&1 && break
    sleep 1
done
if ! curl -s "http://localhost:$PORT/.well-known/agent-card.json" >/dev/null 2>&1; then
    echo "✗ crush-a2a failed to start on port $PORT" >&2
    exit 1
fi
echo "✓ crush-a2a: http://localhost:$PORT"

# ── 4. Run suite ─────────────────────────────────────────────────────
echo "▸ Running A2A v1.0 suite..."
mkdir -p "$REPORTS"

"$BINARY" run "$SPEC" \
    --runtime "$RUNTIME" \
    --url "http://localhost:$PORT" \
    > "$REPORTS/$RUNTIME.md" \
    2>"$REPORTS/$RUNTIME.log"

"$BINARY" run "$SPEC" \
    --runtime "$RUNTIME" \
    --url "http://localhost:$PORT" \
    --format json \
    > "$REPORTS/$RUNTIME.json" \
    2>"$REPORTS/$RUNTIME-json.log"

# ── 5. Report ────────────────────────────────────────────────────────
SCORE=$(grep 'Compliance' "$REPORTS/$RUNTIME.md" | grep -o '[0-9]*\.[0-9]*%' || echo "?")
echo "✓ $RUNTIME: $SCORE"
echo "  Reports: $REPORTS/$RUNTIME.md, $REPORTS/$RUNTIME.json"
