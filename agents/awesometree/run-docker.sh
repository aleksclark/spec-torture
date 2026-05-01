#!/usr/bin/env bash
#
# Run the full ARP compliance suite in Docker.
#
# Starts awesometree + echo-agent + crush-agent (with real bedrock credentials),
# then runs the HTTP test suite against the containerized stack.
# gRPC tests run separately via run-grpc.sh against the live daemon.
#
# Usage: run-docker.sh <reports-dir> <spec-torture-binary>
#
# Prerequisites:
#   - Docker + docker compose
#   - AWS_BEARER_TOKEN_BEDROCK set in environment
#   - crush-a2a binary at /tmp/crush-a2a/crush (or CRUSH_A2A_BIN override)
#
set -euo pipefail

REPORTS="${1:?Usage: run-docker.sh <reports-dir> <spec-torture-binary>}"
BINARY="${2:?}"
DIR="$(cd "$(dirname "$0")" && pwd)"
SPEC="$DIR/../../specs/arp/spec-http.yaml"
RUNTIME="awesometree-docker"
CRUSH_BIN="${CRUSH_A2A_BIN:-/tmp/crush-a2a/crush}"
HTTP_PORT=19099

# ── 1. Preflight ─────────────────────────────────────────────────────
if [ -z "${AWS_BEARER_TOKEN_BEDROCK:-}" ]; then
    echo "✗ AWS_BEARER_TOKEN_BEDROCK not set" >&2
    exit 1
fi

if [ ! -f "$CRUSH_BIN" ]; then
    echo "✗ crush binary not found at $CRUSH_BIN" >&2
    echo "  Download from: https://github.com/aleksclark/crush-modules/releases" >&2
    exit 1
fi

# Copy binaries into docker build context
cp "$CRUSH_BIN" "$DIR/docker/crush"

AWESOMETREE_BIN="${AWESOMETREE_BIN:-/home/aleks/work/projects/awesometree/repo/target/release/arp-test-server}"
if [ ! -f "$AWESOMETREE_BIN" ]; then
    echo "✗ arp-test-server binary not found at $AWESOMETREE_BIN" >&2
    echo "  Build awesometree first: cd awesometree && cargo build --release --bin arp-test-server" >&2
    exit 1
fi
cp "$AWESOMETREE_BIN" "$DIR/docker/arp-test-server"

echo "▸ crush binary: $($CRUSH_BIN --version 2>&1 || echo 'unknown')"

# ── 2. Build and start ───────────────────────────────────────────────
echo "▸ Starting Docker stack..."
docker compose -f "$DIR/docker-compose.yml" up -d --build --wait 2>&1 | tail -5

# Verify services are up
echo "▸ Waiting for services..."
for i in $(seq 1 30); do
    if python3 -c "
import urllib.request
urllib.request.urlopen('http://localhost:$HTTP_PORT/a2a/agents', timeout=2)
" 2>/dev/null; then
        break
    fi
    sleep 2
done

# Check agent count
AGENT_COUNT=$(python3 -c "
import urllib.request, json
resp = urllib.request.urlopen('http://localhost:$HTTP_PORT/a2a/agents')
agents = json.loads(resp.read())
print(len(agents))
for a in agents:
    name = a.get('name', '?')
    arp = a.get('metadata', {}).get('arp', {})
    status = arp.get('status', '?')
    print(f'  {name}: {status}')
" 2>&1)
echo "✓ ARP stack running: $AGENT_COUNT"

# ── 3. Run HTTP suite ────────────────────────────────────────────────
echo ""
echo "▸ Running ARP HTTP suite..."
mkdir -p "$REPORTS"

"$BINARY" run "$SPEC" \
    --runtime "$RUNTIME" \
    --url "http://localhost:$HTTP_PORT" \
    > "$REPORTS/$RUNTIME.md" \
    2>"$REPORTS/$RUNTIME.log" || true

"$BINARY" run "$SPEC" \
    --runtime "$RUNTIME" \
    --url "http://localhost:$HTTP_PORT" \
    --format json \
    > "$REPORTS/$RUNTIME.json" \
    2>"$REPORTS/$RUNTIME-json.log" || true

HTTP_SCORE=$(grep 'Compliance' "$REPORTS/$RUNTIME.md" | grep -o '[0-9]*\.[0-9]*%' || echo "?")
echo "✓ HTTP: $HTTP_SCORE"

# ── 4. Report ────────────────────────────────────────────────────────
echo ""
echo "Reports: $REPORTS/$RUNTIME.md"

# ── 6. Cleanup ───────────────────────────────────────────────────────
echo ""
echo "▸ Stopping Docker stack..."
docker compose -f "$DIR/docker-compose.yml" down 2>&1 | tail -3
rm -f "$DIR/docker/crush" "$DIR/docker/arp-test-server"
