#!/usr/bin/env bash
#
# Run A2A + ARP client conformance tests against hermes-agent's tool handlers.
#
# Usage: run.sh <reports-dir> <a2a-mock-binary>
#
set -euo pipefail

REPORTS="${1:?Usage: run.sh <reports-dir> <a2a-mock-binary>}"
MOCK_BIN="${2:?}"
DIR="$(cd "$(dirname "$0")" && pwd)"
PIDS=()

cleanup() {
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    done
}
trap cleanup EXIT

# ── 1. Check deps ─────────────────────────────────────────────────────
if [ ! -x "$MOCK_BIN" ]; then
    echo "✗ mock binary not found: $MOCK_BIN" >&2
    exit 1
fi

HERMES_ROOT="$DIR/.build/hermes-agent"
if [ ! -d "$HERMES_ROOT" ]; then
    echo "▸ Cloning hermes-agent..."
    mkdir -p "$DIR/.build"
    git clone --depth 1 git@github.com:aleksclark/hermes-agent.git "$HERMES_ROOT"
else
    echo "▸ Updating hermes-agent..."
    git -C "$HERMES_ROOT" pull --ff-only 2>/dev/null || true
fi

# Install httpx if needed
python3 -c "import httpx" 2>/dev/null || pip3 install --quiet httpx

# ── 2. Start mock server ─────────────────────────────────────────────
MOCK_OUTPUT=$(mktemp)
"$MOCK_BIN" > "$MOCK_OUTPUT" &
PIDS+=($!)
sleep 1

if ! kill -0 "${PIDS[-1]}" 2>/dev/null; then
    echo "✗ mock server failed to start" >&2
    exit 1
fi

MOCK_URL=$(python3 -c "import json; print(json.load(open('$MOCK_OUTPUT'))['url'])")
echo "✓ mock server: $MOCK_URL"

# ── 3. Run tests ─────────────────────────────────────────────────────
mkdir -p "$REPORTS"

echo "▸ Running hermes A2A + ARP client suite..."
python3 "$DIR/test_hermes_clients.py" \
    --mock-url "$MOCK_URL" \
    > "$REPORTS/hermes-agent-client.md" || true

python3 "$DIR/test_hermes_clients.py" \
    --mock-url "$MOCK_URL" \
    --json \
    > "$REPORTS/hermes-agent-client.json" || true

# ── 4. Report ────────────────────────────────────────────────────────
SCORE=$(grep 'Compliance' "$REPORTS/hermes-agent-client.md" | grep -o '[0-9]*\.[0-9]*%' || echo "?")
echo "✓ hermes-agent: $SCORE"
echo "  Reports: $REPORTS/hermes-agent-client.md, $REPORTS/hermes-agent-client.json"

rm -f "$MOCK_OUTPUT"
