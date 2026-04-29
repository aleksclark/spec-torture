#!/usr/bin/env bash
#
# Run the ARP HTTP spec-torture suite against a live awesometree-daemon.
#
# Usage: run.sh <spec-yaml> <reports-dir> <spec-torture-binary>
#
# Prerequisites:
#   - awesometree-daemon running on localhost:9099
#   - At least one workspace with an active agent (for proxy chained tests)
#
set -euo pipefail

SPEC="${1:?Usage: run.sh <spec> <reports-dir> <binary>}"
REPORTS="${2:?}"
BINARY="${3:?}"
RUNTIME="awesometree"
URL="${AWESOMETREE_URL:-http://localhost:9099}"
VERSION="${AWESOMETREE_VERSION:-0.1.0}"

# ── 1. Check awesometree-daemon is running ──────────────────────────
if ! pgrep -f awesometree-daemon >/dev/null 2>&1; then
    echo "✗ awesometree-daemon is not running" >&2
    echo "  Start it with: awesometree-daemon" >&2
    exit 1
fi
echo "✓ awesometree-daemon: running"

# ── 2. Verify HTTP connectivity ─────────────────────────────────────
if ! "$BINARY" run "$SPEC" --url "$URL" --runtime "$RUNTIME" --dry-run >/dev/null 2>&1; then
    # Fallback: just check the workspaces endpoint responds
    HTTP_STATUS=$(curl -s -o /dev/null -w '%{http_code}' "$URL/api/workspaces" 2>/dev/null || echo "000")
    if [ "$HTTP_STATUS" = "000" ]; then
        echo "✗ Cannot reach awesometree at $URL" >&2
        exit 1
    fi
fi
echo "✓ awesometree: $URL"

# ── 3. Check for agents (informational) ─────────────────────────────
AGENT_STATUS=$(curl -s -o /dev/null -w '%{http_code}' "$URL/a2a/agents" 2>/dev/null || echo "000")
if [ "$AGENT_STATUS" = "200" ]; then
    AGENT_COUNT=$(curl -s "$URL/a2a/agents" 2>/dev/null | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
    echo "✓ ARP proxy: $AGENT_COUNT agents available"
else
    echo "⚠ ARP proxy: /a2a/agents returned $AGENT_STATUS (proxy not yet implemented)"
fi

# ── 4. Run suite ─────────────────────────────────────────────────────
echo "▸ Running ARP HTTP suite..."
mkdir -p "$REPORTS"

"$BINARY" run "$SPEC" \
    --runtime "$RUNTIME" \
    --runtime-version "$VERSION" \
    --url "$URL" \
    > "$REPORTS/$RUNTIME.md" \
    2>"$REPORTS/$RUNTIME.log"

"$BINARY" run "$SPEC" \
    --runtime "$RUNTIME" \
    --runtime-version "$VERSION" \
    --url "$URL" \
    --format json \
    > "$REPORTS/$RUNTIME.json" \
    2>"$REPORTS/$RUNTIME-json.log"

# ── 5. Report ────────────────────────────────────────────────────────
SCORE=$(grep 'Compliance' "$REPORTS/$RUNTIME.md" | grep -o '[0-9]*\.[0-9]*%' || echo "?")
echo "✓ $RUNTIME: $SCORE"
echo "  Reports: $REPORTS/$RUNTIME.md, $REPORTS/$RUNTIME.json"
