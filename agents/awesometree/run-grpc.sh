#!/usr/bin/env bash
#
# ARP gRPC compliance test suite.
# Tests all 5 ARP services against a live awesometree-daemon.
#
# Usage: run-grpc.sh [host:port]
#
set -euo pipefail

ADDR="${1:-localhost:9098}"
GRPCURL="${GRPCURL:-/tmp/grpcurl/grpcurl}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROTO_DIR="$SCRIPT_DIR/../../proto"

PASS=0
FAIL=0
ERRORS=""

grpc_call() {
    local method="$1"
    shift
    "$GRPCURL" -plaintext \
        -import-path "$PROTO_DIR" \
        -import-path /usr/include \
        -proto arp/v1/project.proto \
        -proto arp/v1/workspace.proto \
        -proto arp/v1/agent.proto \
        -proto arp/v1/discovery.proto \
        -proto arp/v1/token.proto \
        "$@" "$ADDR" "$method" 2>&1
}

run_test() {
    local id="$1" name="$2" severity="$3"
    shift 3
    local output
    if output=$("$@" 2>&1); then
        PASS=$((PASS + 1))
        printf "  PASS  %-55s %s\n" "$id" "$severity"
    else
        FAIL=$((FAIL + 1))
        local err_line
        err_line=$(echo "$output" | head -1)
        ERRORS="${ERRORS}\n  FAIL  $id: $err_line"
        printf "  FAIL  %-55s %s  %s\n" "$id" "$severity" "$err_line"
    fi
}

# Test: expect gRPC OK (no error code in output)
test_ok() {
    local method="$1"
    shift
    local output
    output=$(grpc_call "$method" "$@" 2>&1)
    local rc=$?
    if [ $rc -eq 0 ]; then
        return 0
    fi
    echo "$output" | head -1
    return 1
}

# Test: expect specific gRPC status code
test_status() {
    local expected="$1" method="$2"
    shift 2
    local output
    output=$(grpc_call "$method" "$@" 2>&1) || true
    if echo "$output" | grep -q "Code: $expected"; then
        return 0
    fi
    echo "expected $expected, got: $(echo "$output" | grep 'Code:' | head -1)"
    return 1
}

# Test: output contains a string
test_contains() {
    local needle="$1" method="$2"
    shift 2
    local output
    output=$(grpc_call "$method" "$@" 2>&1)
    if echo "$output" | grep -q "$needle"; then
        return 0
    fi
    echo "expected output to contain '$needle'"
    return 1
}

# Test: output does NOT contain a string
test_not_contains() {
    local needle="$1" method="$2"
    shift 2
    local output
    output=$(grpc_call "$method" "$@" 2>&1)
    if echo "$output" | grep -q "$needle"; then
        echo "output should not contain '$needle'"
        return 1
    fi
    return 0
}

# Test: capture output and extract a JSON field
grpc_field() {
    local method="$1" field="$2"
    shift 2
    local output
    output=$(grpc_call "$method" "$@" 2>&1) || return 1
    echo "$output" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d$(echo "$field" | sed 's/\./][\"/' | sed 's/^/[\"/' | sed 's/$/\"]/'))" 2>/dev/null
}

echo "# ARP gRPC Compliance: $ADDR"
echo ""

# =========================================================================
echo "## ProjectService"
# =========================================================================

run_test "project-list" \
    "ListProjects returns OK" required \
    test_ok arp.v1.ProjectService/ListProjects

run_test "project-list-response" \
    "ListProjects returns projects array" required \
    test_contains "projects" arp.v1.ProjectService/ListProjects

# =========================================================================
echo ""
echo "## WorkspaceService"
# =========================================================================

run_test "workspace-list" \
    "ListWorkspaces returns OK" required \
    test_ok arp.v1.WorkspaceService/ListWorkspaces

run_test "workspace-list-filter-project" \
    "ListWorkspaces accepts project filter" required \
    test_ok arp.v1.WorkspaceService/ListWorkspaces -d '{"project":"nonexistent-00000"}'

run_test "workspace-get-nonexistent" \
    "GetWorkspace returns NotFound for unknown" required \
    test_status NotFound arp.v1.WorkspaceService/GetWorkspace -d '{"name":"nonexistent-ws-00000"}'

run_test "workspace-destroy-nonexistent" \
    "DestroyWorkspace returns NotFound for unknown" required \
    test_status NotFound arp.v1.WorkspaceService/DestroyWorkspace -d '{"name":"nonexistent-ws-00000"}'

# =========================================================================
echo ""
echo "## AgentService"
# =========================================================================

run_test "agent-list" \
    "ListAgents returns OK" required \
    test_ok arp.v1.AgentService/ListAgents

run_test "agent-list-filter-workspace" \
    "ListAgents accepts workspace filter" required \
    test_ok arp.v1.AgentService/ListAgents -d '{"workspace":"nonexistent-00000"}'

run_test "agent-list-filter-status" \
    "ListAgents accepts status filter" required \
    test_ok arp.v1.AgentService/ListAgents -d '{"status":"AGENT_STATUS_READY"}'

run_test "agent-get-nonexistent" \
    "GetAgentStatus returns NotFound for unknown" required \
    test_status NotFound arp.v1.AgentService/GetAgentStatus -d '{"agentId":"nonexistent-00000"}'

run_test "agent-stop-nonexistent" \
    "StopAgent returns NotFound for unknown" required \
    test_status NotFound arp.v1.AgentService/StopAgent -d '{"agentId":"nonexistent-00000"}'

run_test "agent-restart-nonexistent" \
    "RestartAgent returns NotFound for unknown" required \
    test_status NotFound arp.v1.AgentService/RestartAgent -d '{"agentId":"nonexistent-00000"}'

run_test "agent-message-nonexistent" \
    "SendAgentMessage returns NotFound for unknown" required \
    test_status NotFound arp.v1.AgentService/SendAgentMessage -d '{"agentId":"nonexistent-00000","message":"hi"}'

run_test "agent-task-status-nonexistent" \
    "GetAgentTaskStatus returns NotFound for unknown" required \
    test_status NotFound arp.v1.AgentService/GetAgentTaskStatus -d '{"agentId":"nonexistent-00000","taskId":"fake"}'

run_test "agent-spawn-missing-workspace" \
    "SpawnAgent returns InvalidArgument without workspace" required \
    test_status InvalidArgument arp.v1.AgentService/SpawnAgent -d '{"template":"crush"}'

run_test "agent-spawn-missing-template" \
    "SpawnAgent returns InvalidArgument without template" required \
    test_status InvalidArgument arp.v1.AgentService/SpawnAgent -d '{"workspace":"test"}'

# =========================================================================
echo ""
echo "## DiscoveryService"
# =========================================================================

run_test "discover-agents" \
    "DiscoverAgents returns OK" required \
    test_ok arp.v1.DiscoveryService/DiscoverAgents

run_test "discover-local-scope" \
    "DiscoverAgents with LOCAL scope" required \
    test_ok arp.v1.DiscoveryService/DiscoverAgents -d '{"scope":"DISCOVERY_SCOPE_LOCAL"}'

run_test "discover-capability-filter" \
    "DiscoverAgents with capability filter" required \
    test_ok arp.v1.DiscoveryService/DiscoverAgents -d '{"capability":"nonexistent"}'

run_test "watch-agent-nonexistent" \
    "WatchAgent returns NotFound for unknown" required \
    test_status NotFound arp.v1.DiscoveryService/WatchAgent -d '{"agentId":"nonexistent-00000"}'

# =========================================================================
echo ""
echo "## Lifecycle (echo-agent)"
echo "## Requires running echo-agent-001 in arp-test workspace"
# =========================================================================

# -- Agent status for a known agent --
run_test "lifecycle-get-echo-agent" \
    "GetAgentStatus returns echo-agent-001" required \
    test_ok arp.v1.AgentService/GetAgentStatus -d '{"agentId":"echo-agent-001"}'

run_test "lifecycle-echo-agent-status-ready" \
    "echo-agent-001 status is AGENT_STATUS_READY" required \
    test_contains "AGENT_STATUS_READY" arp.v1.AgentService/GetAgentStatus -d '{"agentId":"echo-agent-001"}'

run_test "lifecycle-echo-agent-has-template" \
    "echo-agent-001 response has template field" required \
    test_contains "template" arp.v1.AgentService/GetAgentStatus -d '{"agentId":"echo-agent-001"}'

run_test "lifecycle-echo-agent-has-workspace" \
    "echo-agent-001 response has workspace field" required \
    test_contains "workspace" arp.v1.AgentService/GetAgentStatus -d '{"agentId":"echo-agent-001"}'

run_test "lifecycle-echo-agent-has-port" \
    "echo-agent-001 response has port field" required \
    test_contains "port" arp.v1.AgentService/GetAgentStatus -d '{"agentId":"echo-agent-001"}'

# -- ListAgents shows the echo agent --
run_test "lifecycle-list-shows-echo" \
    "ListAgents includes echo-agent-001" required \
    test_contains "echo-agent-001" arp.v1.AgentService/ListAgents

run_test "lifecycle-list-filter-ready" \
    "ListAgents status=READY includes echo-agent" required \
    test_contains "echo-agent" arp.v1.AgentService/ListAgents -d '{"status":"AGENT_STATUS_READY"}'

run_test "lifecycle-list-filter-workspace" \
    "ListAgents workspace=arp-test includes echo-agent" required \
    test_contains "echo-agent" arp.v1.AgentService/ListAgents -d '{"workspace":"arp-test"}'

# -- Send a message via gRPC to echo-agent (proxied through ARP) --
# NOTE: echo-agent speaks HTTP+JSON, not JSON-RPC. The gRPC proxy sends
# JSON-RPC to agents, so this correctly returns Internal (404 from agent).
# This validates the proxy error propagation path.
run_test "lifecycle-send-message-echo-error-propagation" \
    "SendAgentMessage to HTTP+JSON agent returns Internal (not crash)" required \
    test_status Internal arp.v1.AgentService/SendAgentMessage -d '{"agentId":"echo-agent-001","message":"hello from grpc test"}'

# -- Discovery finds the echo agent --
run_test "lifecycle-discover-finds-echo" \
    "DiscoverAgents includes echo-agent" required \
    test_contains "echo-agent" arp.v1.DiscoveryService/DiscoverAgents

run_test "lifecycle-discover-by-echo-capability" \
    "DiscoverAgents capability=echo finds echo-agent" required \
    test_contains "echo-agent" arp.v1.DiscoveryService/DiscoverAgents -d '{"capability":"echo"}'

run_test "lifecycle-discover-no-match" \
    "DiscoverAgents capability=nonexistent excludes echo-agent" required \
    test_not_contains "echo-agent" arp.v1.DiscoveryService/DiscoverAgents -d '{"capability":"nonexistent-00000"}'

# -- GetWorkspace shows agents --
run_test "lifecycle-workspace-has-agents" \
    "GetWorkspace arp-test includes agents" required \
    test_contains "agents" arp.v1.WorkspaceService/GetWorkspace -d '{"name":"arp-test"}'

run_test "lifecycle-workspace-is-active" \
    "GetWorkspace arp-test status is ACTIVE" required \
    test_contains "WORKSPACE_STATUS_ACTIVE" arp.v1.WorkspaceService/GetWorkspace -d '{"name":"arp-test"}'

# =========================================================================
echo ""
echo "## Lifecycle (crush-agent)"
echo "## Requires running crush-agent-001 in arp-test workspace"
# =========================================================================

run_test "lifecycle-crush-agent-ready" \
    "GetAgentStatus crush-agent-001 is READY" required \
    test_contains "AGENT_STATUS_READY" arp.v1.AgentService/GetAgentStatus -d '{"agentId":"crush-agent-001"}'

run_test "lifecycle-send-message-crush" \
    "SendAgentMessage to crush-agent-001 returns OK" required \
    test_ok arp.v1.AgentService/SendAgentMessage -d '{"agentId":"crush-agent-001","message":"say hello in one word"}'

run_test "lifecycle-crush-message-has-result" \
    "SendAgentMessage response has task or message" required \
    test_contains "task\|message" arp.v1.AgentService/SendAgentMessage -d '{"agentId":"crush-agent-001","message":"say hi"}'

run_test "lifecycle-crush-message-blocking" \
    "SendAgentMessage blocking=true to crush returns OK" required \
    test_ok arp.v1.AgentService/SendAgentMessage -d '{"agentId":"crush-agent-001","message":"say ok","blocking":true}'

run_test "lifecycle-discover-finds-crush" \
    "DiscoverAgents includes crush-agent" required \
    test_contains "crush-agent" arp.v1.DiscoveryService/DiscoverAgents

run_test "lifecycle-discover-by-crush-capability" \
    "DiscoverAgents capability=crush finds crush-agent" required \
    test_contains "crush-agent" arp.v1.DiscoveryService/DiscoverAgents -d '{"capability":"crush"}'

# =========================================================================
echo ""
echo "## TokenService"
# =========================================================================

run_test "token-create" \
    "CreateToken returns OK" required \
    test_ok arp.v1.TokenService/CreateToken -d '{"subject":"test","scope":{"global":true},"permission":"PERMISSION_ADMIN"}'

run_test "token-create-has-bearer" \
    "CreateToken returns bearerToken" required \
    test_contains "bearerToken" arp.v1.TokenService/CreateToken -d '{"subject":"test","scope":{"global":true},"permission":"PERMISSION_ADMIN"}'

run_test "token-create-missing-subject" \
    "CreateToken returns InvalidArgument without subject" required \
    test_status InvalidArgument arp.v1.TokenService/CreateToken -d '{"scope":{"global":true},"permission":"PERMISSION_ADMIN"}'

run_test "token-create-missing-scope" \
    "CreateToken returns InvalidArgument without scope" required \
    test_status InvalidArgument arp.v1.TokenService/CreateToken -d '{"subject":"test","permission":"PERMISSION_ADMIN"}'

# =========================================================================
echo ""
# =========================================================================

TOTAL=$((PASS + FAIL))
if [ "$TOTAL" -eq 0 ]; then
    echo "No tests ran"
    exit 1
fi
COMPLIANCE=$(python3 -c "print(f'{$PASS/$TOTAL*100:.1f}')")

echo "## Summary"
echo ""
echo "| Metric | Count |"
echo "|--------|-------|"
echo "| Total | $TOTAL |"
echo "| Passed | $PASS |"
echo "| Failed | $FAIL |"
echo "| **Compliance** | **${COMPLIANCE}%** |"

if [ -n "$ERRORS" ]; then
    echo ""
    echo "## Failures"
    echo -e "$ERRORS"
fi

[ "$FAIL" -eq 0 ]
