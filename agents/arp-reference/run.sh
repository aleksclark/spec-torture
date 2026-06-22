#!/usr/bin/env bash
#
# Reference ARP implementation conformance runner.
#
# Builds the reference server + conformance tool, boots the server (seeded,
# in-process mock agent backend) on a free port, then:
#
#   1. Runs the Go conformance suite       -> reports/arp/reference-grpc.md
#   2. Runs the grpcurl suite (reflection) -> reports/arp/reference-grpcurl.md
#
# Usage: run.sh [reports_dir]
#
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPORTS="${1:-$ROOT/reports/arp}"

export PATH="$(go env GOPATH 2>/dev/null)/bin:$PATH"
GRPCURL="${GRPCURL:-grpcurl}"

mkdir -p "$REPORTS" "$ROOT/bin"

echo "==> Building reference binaries"
( cd "$ROOT" && go build -o bin/arp-server ./cmd/arp-server && go build -o bin/arp-conformance ./cmd/arp-conformance ) || {
    echo "build failed" >&2
    exit 1
}

# Pick a free TCP port.
PORT="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"
ADDR="127.0.0.1:$PORT"

echo "==> Starting reference arp-server (seeded) on $ADDR"
"$ROOT/bin/arp-server" -seed -addr "$ADDR" >/tmp/arp-reference-server.log 2>&1 &
SRV=$!
cleanup() { kill "$SRV" 2>/dev/null || true; }
trap cleanup EXIT

# Wait for readiness via reflection.
ready=0
for _ in $(seq 1 50); do
    if "$GRPCURL" -plaintext "$ADDR" list >/dev/null 2>&1; then
        ready=1
        break
    fi
    sleep 0.1
done
if [ "$ready" -ne 1 ]; then
    echo "server did not become ready; log:" >&2
    cat /tmp/arp-reference-server.log >&2
    exit 1
fi

echo "==> Go conformance suite -> $REPORTS/reference-grpc.md"
"$ROOT/bin/arp-conformance" -out "$REPORTS/reference-grpc.md" >/dev/null
GO_RC=$?

echo "==> grpcurl suite (reflection) -> $REPORTS/reference-grpcurl.md"
"$SCRIPT_DIR/run-grpc.sh" "$ADDR" | tee "$REPORTS/reference-grpcurl.md"
GRPCURL_RC=${PIPESTATUS[0]}

echo ""
echo "Go conformance exit=$GO_RC, grpcurl suite exit=$GRPCURL_RC"
[ "$GO_RC" -eq 0 ] && [ "$GRPCURL_RC" -eq 0 ]
