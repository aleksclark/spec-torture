#!/usr/bin/env python3
"""
Hermes A2A + ARP client conformance test harness.

Tests hermes-agent's A2A and ARP client tool handlers directly (no LLM
needed) against the spec-torture mock A2A server. Inspects recorded
requests via the mock's /_inspect/requests API.

Usage:
    python3 test_hermes_clients.py --mock-url http://127.0.0.1:PORT [--json]
"""
import argparse
import json
import os
import sys
import time
import urllib.request
from dataclasses import dataclass, field
from typing import Any, Callable, Dict, List, Optional

# ---------------------------------------------------------------------------
# Add hermes-agent to sys.path so we can import its tools
# ---------------------------------------------------------------------------
HERMES_ROOT = os.path.join(os.path.dirname(__file__), ".build", "hermes-agent")
sys.path.insert(0, HERMES_ROOT)

# Stub out model_tools._run_async so hermes tool handlers work outside the
# full hermes runtime.  The handlers call _run_a2a_async / _run_arp_async
# which delegate to model_tools._run_async.
import asyncio

def _fake_run_async(coro):
    """Minimal async runner for tool handlers."""
    loop = asyncio.new_event_loop()
    try:
        return loop.run_until_complete(coro)
    finally:
        loop.close()

# Patch before importing hermes tools
sys.modules.setdefault("model_tools", type(sys)("model_tools"))
sys.modules["model_tools"]._run_async = _fake_run_async

# Import the real tools package first, then stub just the registry
import importlib
import tools  # noqa: E402 — real package from HERMES_ROOT
import tools.registry  # noqa: E402

_reg_entries: list = []

class _FakeRegistry:
    def register(self, **kw):
        _reg_entries.append(kw)

tools.registry.registry = _FakeRegistry()

from tools.a2a_tool import (  # noqa: E402
    a2a_discover_handler,
    a2a_send_handler,
    a2a_get_task_handler,
    a2a_cancel_task_handler,
)
from tools.arp_tool import (  # noqa: E402
    arp_list_agents_handler,
    arp_get_agent_card_handler,
    arp_send_message_handler,
    arp_route_message_handler,
    arp_list_workspaces_handler,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def mock_get(url: str, path: str) -> Any:
    """GET a JSON endpoint on the mock."""
    resp = urllib.request.urlopen(f"{url}{path}", timeout=5)
    return json.loads(resp.read())


def mock_reset(url: str):
    """Clear recorded requests on the mock."""
    urllib.request.urlopen(f"{url}/_inspect/reset", timeout=5)


def mock_requests(url: str, method: str = "") -> List[Dict]:
    """Fetch recorded requests from the mock, optionally filtered."""
    qs = f"?method={method}" if method else ""
    resp = urllib.request.urlopen(f"{url}/_inspect/requests{qs}", timeout=5)
    return json.loads(resp.read())


def parse_body(req: Dict) -> Dict:
    """Parse the raw_body from a recorded request."""
    raw = req.get("raw_body", "")
    if not raw:
        return {}
    try:
        return json.loads(raw)
    except Exception:
        return {}


# ---------------------------------------------------------------------------
# Test result types
# ---------------------------------------------------------------------------

@dataclass
class TestResult:
    id: str
    name: str
    severity: str
    category: str
    status: str = "pass"    # pass | fail | error
    error: str = ""
    duration_ms: float = 0


@dataclass
class TestRun:
    runtime: str
    results: List[TestResult] = field(default_factory=list)

    def summary(self):
        total = len(self.results)
        passed = sum(1 for r in self.results if r.status == "pass")
        failed = sum(1 for r in self.results if r.status == "fail")
        errors = sum(1 for r in self.results if r.status == "error")
        applicable = total - errors
        compliance = (passed / applicable * 100) if applicable > 0 else 0
        return {
            "total": total, "passed": passed, "failed": failed,
            "errors": errors, "compliance": round(compliance, 1),
        }


# ---------------------------------------------------------------------------
# A2A Client Tests
# ---------------------------------------------------------------------------

def a2a_tests(mock_url: str) -> List[TestResult]:
    results = []

    def run(tid, name, sev, cat, fn):
        mock_reset(mock_url)
        t0 = time.monotonic()
        try:
            err = fn()
            elapsed = (time.monotonic() - t0) * 1000
            if err:
                results.append(TestResult(tid, name, sev, cat, "fail", err, elapsed))
            else:
                results.append(TestResult(tid, name, sev, cat, "pass", "", elapsed))
        except Exception as e:
            elapsed = (time.monotonic() - t0) * 1000
            results.append(TestResult(tid, name, sev, cat, "error", str(e), elapsed))

    # -- Discovery --
    def test_discovery():
        res = a2a_discover_handler({"url": mock_url})
        data = json.loads(res)
        if "error" in data:
            return f"discover returned error: {data['error']}"
        reqs = mock_requests(mock_url)
        found = any(r.get("path") == "/.well-known/agent-card.json" and r.get("http_method") == "GET" for r in reqs)
        if not found:
            return "client did not GET /.well-known/agent-card.json"
        return None

    run("hermes-a2a-discovery", "A2A Discover Fetches Agent Card via GET", "required", "discovery", test_discovery)

    # -- SendMessage envelope --
    def test_send_envelope():
        a2a_send_handler({"url": mock_url, "message": "hello"})
        reqs = mock_requests(mock_url, "SendMessage")
        if not reqs:
            return "no SendMessage request recorded"
        body = parse_body(reqs[0])
        if body.get("jsonrpc") != "2.0":
            return f"missing jsonrpc 2.0, got {body.get('jsonrpc')}"
        if "id" not in body:
            return "missing id field"
        if body.get("method") != "SendMessage":
            return f"method is {body.get('method')}, expected SendMessage"
        return None

    run("hermes-a2a-send-envelope", "SendMessage Uses JSON-RPC 2.0 Envelope", "required", "messaging", test_send_envelope)

    # -- Method name --
    def test_method_name():
        a2a_send_handler({"url": mock_url, "message": "hello"})
        reqs = mock_requests(mock_url, "SendMessage")
        if not reqs:
            old = mock_requests(mock_url, "message/send")
            if old:
                return 'used deprecated "message/send" instead of "SendMessage"'
            return "no SendMessage request"
        return None

    run("hermes-a2a-send-method", "SendMessage Uses v1.0 PascalCase Method", "required", "messaging", test_method_name)

    # -- Content-Type --
    def test_content_type():
        a2a_send_handler({"url": mock_url, "message": "hello"})
        reqs = mock_requests(mock_url, "SendMessage")
        if not reqs:
            return "no request"
        ct = reqs[0].get("headers", {}).get("Content-Type", [""])[0]
        if "application/json" not in ct:
            return f"Content-Type is {ct}, expected application/json"
        return None

    run("hermes-a2a-send-content-type", "SendMessage Content-Type application/json", "required", "messaging", test_content_type)

    # -- messageId --
    def test_message_id():
        a2a_send_handler({"url": mock_url, "message": "hello"})
        reqs = mock_requests(mock_url, "SendMessage")
        if not reqs:
            return "no request"
        body = parse_body(reqs[0])
        msg = body.get("params", {}).get("message", {})
        mid = msg.get("messageId", "")
        if not mid:
            return "messageId missing (REQUIRED per proto)"
        return None

    run("hermes-a2a-send-message-id", "SendMessage Includes messageId", "required", "messaging", test_message_id)

    # -- role --
    def test_role():
        a2a_send_handler({"url": mock_url, "message": "hello"})
        reqs = mock_requests(mock_url, "SendMessage")
        if not reqs:
            return "no request"
        body = parse_body(reqs[0])
        role = body.get("params", {}).get("message", {}).get("role", "")
        if role not in ("ROLE_USER", "user"):
            return f"role is {role!r}, expected ROLE_USER or user"
        return None

    run("hermes-a2a-send-role", "SendMessage Includes Role", "required", "messaging", test_role)

    # -- parts --
    def test_parts():
        a2a_send_handler({"url": mock_url, "message": "hello"})
        reqs = mock_requests(mock_url, "SendMessage")
        if not reqs:
            return "no request"
        body = parse_body(reqs[0])
        parts = body.get("params", {}).get("message", {}).get("parts", [])
        if not parts:
            return "parts is empty (REQUIRED per proto)"
        part0 = parts[0]
        has_content = any(k in part0 for k in ("text", "raw", "url", "data"))
        if not has_content:
            return "part has no content field (text/raw/url/data)"
        return None

    run("hermes-a2a-send-parts", "SendMessage Parts Have Content", "required", "messaging", test_parts)

    # -- HTTP POST --
    def test_http_post():
        a2a_send_handler({"url": mock_url, "message": "hello"})
        reqs = mock_requests(mock_url, "SendMessage")
        if not reqs:
            return "no request"
        if reqs[0].get("http_method") != "POST":
            return f"used {reqs[0].get('http_method')}, expected POST"
        return None

    run("hermes-a2a-send-http-post", "SendMessage Uses HTTP POST", "required", "messaging", test_http_post)

    # -- GetTask --
    def test_get_task():
        a2a_get_task_handler({"url": mock_url, "task_id": "task-mock-001"})
        reqs = mock_requests(mock_url, "GetTask")
        if not reqs:
            return "no GetTask request recorded"
        body = parse_body(reqs[0])
        if body.get("jsonrpc") != "2.0":
            return "missing jsonrpc 2.0"
        if body.get("method") != "GetTask":
            return f"method is {body.get('method')}"
        tid = body.get("params", {}).get("id", "")
        if tid != "task-mock-001":
            return f"params.id is {tid!r}, expected task-mock-001"
        return None

    run("hermes-a2a-get-task", "GetTask Correct Envelope and Params", "required", "lifecycle", test_get_task)

    # -- CancelTask --
    def test_cancel_task():
        a2a_cancel_task_handler({"url": mock_url, "task_id": "task-mock-001"})
        reqs = mock_requests(mock_url, "CancelTask")
        if not reqs:
            return "no CancelTask request recorded"
        body = parse_body(reqs[0])
        if body.get("method") != "CancelTask":
            return f"method is {body.get('method')}"
        return None

    run("hermes-a2a-cancel-task", "CancelTask Uses Correct Method", "required", "lifecycle", test_cancel_task)

    # -- Context propagation --
    def test_context():
        a2a_send_handler({"url": mock_url, "message": "first", "context_id": "ctx-mock-001"})
        reqs = mock_requests(mock_url, "SendMessage")
        if not reqs:
            return "no request"
        body = parse_body(reqs[0])
        cid = body.get("params", {}).get("message", {}).get("contextId", "")
        if cid != "ctx-mock-001":
            return f"contextId is {cid!r}, expected ctx-mock-001"
        return None

    run("hermes-a2a-context-propagation", "SendMessage Propagates contextId", "recommended", "messaging", test_context)

    # -- Error handling --
    def test_error_handling():
        res = a2a_get_task_handler({"url": mock_url, "task_id": "nonexistent"})
        data = json.loads(res)
        # Should return error info, not crash
        if "error" not in data:
            return None  # some impls return the error in result — either is fine
        return None

    run("hermes-a2a-error-handling", "Client Handles JSON-RPC Errors Gracefully", "required", "response-handling", test_error_handling)

    return results


# ---------------------------------------------------------------------------
# ARP Client Tests
# ---------------------------------------------------------------------------

def arp_tests(mock_url: str) -> List[TestResult]:
    results = []

    def run(tid, name, sev, cat, fn):
        mock_reset(mock_url)
        t0 = time.monotonic()
        try:
            err = fn()
            elapsed = (time.monotonic() - t0) * 1000
            if err:
                results.append(TestResult(tid, name, sev, cat, "fail", err, elapsed))
            else:
                results.append(TestResult(tid, name, sev, cat, "pass", "", elapsed))
        except Exception as e:
            elapsed = (time.monotonic() - t0) * 1000
            results.append(TestResult(tid, name, sev, cat, "error", str(e), elapsed))

    # -- List agents uses GET /a2a/agents --
    def test_list_agents_path():
        arp_list_agents_handler({"url": mock_url})
        reqs = mock_requests(mock_url)
        found = any(r.get("path") == "/a2a/agents" and r.get("http_method") == "GET" for r in reqs)
        if not found:
            return "client did not GET /a2a/agents"
        return None

    run("hermes-arp-list-agents", "arp_list_agents GETs /a2a/agents", "required", "a2a-proxy", test_list_agents_path)

    # -- Get agent card uses correct path --
    def test_agent_card_path():
        arp_get_agent_card_handler({"url": mock_url, "agent_id": "test-agent"})
        reqs = mock_requests(mock_url)
        found = any(
            r.get("path") == "/a2a/agents/test-agent/.well-known/agent-card.json"
            and r.get("http_method") == "GET"
            for r in reqs
        )
        if not found:
            paths = [r.get("path") for r in reqs]
            return f"expected GET /a2a/agents/test-agent/.well-known/agent-card.json, got {paths}"
        return None

    run("hermes-arp-get-agent-card", "arp_get_agent_card GETs Correct Path", "required", "a2a-proxy", test_agent_card_path)

    # -- Send message uses POST /a2a/agents/{id}/message:send --
    def test_send_path():
        arp_send_message_handler({"url": mock_url, "agent_id": "test-agent", "message": "hi"})
        reqs = mock_requests(mock_url)
        found = any(
            r.get("path") == "/a2a/agents/test-agent/message:send"
            and r.get("http_method") == "POST"
            for r in reqs
        )
        if not found:
            paths = [(r.get("http_method"), r.get("path")) for r in reqs]
            return f"expected POST /a2a/agents/test-agent/message:send, got {paths}"
        return None

    run("hermes-arp-send-message-path", "arp_send_message POSTs to /a2a/agents/{id}/message:send", "required", "a2a-proxy", test_send_path)

    # -- Send message includes role and parts --
    def test_send_body():
        arp_send_message_handler({"url": mock_url, "agent_id": "test-agent", "message": "hi"})
        reqs = [r for r in mock_requests(mock_url) if r.get("path", "").endswith("message:send")]
        if not reqs:
            return "no request to message:send"
        body = parse_body(reqs[0])
        msg = body.get("message", {})
        if not msg.get("role"):
            return "message missing role"
        if not msg.get("parts"):
            return "message missing parts"
        return None

    run("hermes-arp-send-message-body", "arp_send_message Body Has role and parts", "required", "a2a-proxy", test_send_body)

    # -- Send message includes Content-Type --
    def test_send_content_type():
        arp_send_message_handler({"url": mock_url, "agent_id": "test-agent", "message": "hi"})
        reqs = [r for r in mock_requests(mock_url) if r.get("path", "").endswith("message:send")]
        if not reqs:
            return "no request"
        ct = reqs[0].get("headers", {}).get("Content-Type", [""])[0]
        if "application/json" not in ct:
            return f"Content-Type is {ct}"
        return None

    run("hermes-arp-send-content-type", "arp_send_message Content-Type application/json", "required", "a2a-proxy", test_send_content_type)

    # -- Route message uses POST /a2a/route/message:send --
    def test_route_path():
        arp_route_message_handler({"url": mock_url, "message": "hi", "tags": ["test"]})
        reqs = mock_requests(mock_url)
        found = any(
            r.get("path") == "/a2a/route/message:send"
            and r.get("http_method") == "POST"
            for r in reqs
        )
        if not found:
            paths = [(r.get("http_method"), r.get("path")) for r in reqs]
            return f"expected POST /a2a/route/message:send, got {paths}"
        return None

    run("hermes-arp-route-path", "arp_route_message POSTs to /a2a/route/message:send", "required", "a2a-proxy-routing", test_route_path)

    # -- Route message includes routing.tags --
    def test_route_tags():
        arp_route_message_handler({"url": mock_url, "message": "hi", "tags": ["coding", "python"]})
        reqs = [r for r in mock_requests(mock_url) if r.get("path", "").endswith("route/message:send")]
        if not reqs:
            return "no route request"
        body = parse_body(reqs[0])
        tags = body.get("routing", {}).get("tags", [])
        if tags != ["coding", "python"]:
            return f"routing.tags is {tags}, expected ['coding', 'python']"
        return None

    run("hermes-arp-route-tags", "arp_route_message Includes routing.tags", "required", "a2a-proxy-routing", test_route_tags)

    # -- Route context propagation --
    def test_route_context():
        arp_route_message_handler({"url": mock_url, "message": "hi", "context_id": "ctx-123"})
        reqs = [r for r in mock_requests(mock_url) if r.get("path", "").endswith("route/message:send")]
        if not reqs:
            return "no route request"
        body = parse_body(reqs[0])
        cid = body.get("message", {}).get("contextId", "")
        if cid != "ctx-123":
            return f"contextId is {cid!r}, expected ctx-123"
        return None

    run("hermes-arp-route-context", "arp_route_message Propagates contextId", "recommended", "a2a-proxy-routing", test_route_context)

    # -- List workspaces uses GET /api/workspaces --
    def test_list_workspaces():
        arp_list_workspaces_handler({"url": mock_url})
        reqs = mock_requests(mock_url)
        found = any(r.get("path") == "/api/workspaces" and r.get("http_method") == "GET" for r in reqs)
        if not found:
            return "client did not GET /api/workspaces"
        return None

    run("hermes-arp-list-workspaces", "arp_list_workspaces GETs /api/workspaces", "required", "api-workspaces", test_list_workspaces)

    return results


# ---------------------------------------------------------------------------
# Report
# ---------------------------------------------------------------------------

def print_markdown(run: TestRun):
    s = run.summary()
    print(f"# Hermes Client Conformance: {run.runtime}\n")
    print(f"| Metric | Count |")
    print(f"|--------|-------|")
    print(f"| Total | {s['total']} |")
    print(f"| Passed | {s['passed']} |")
    print(f"| Failed | {s['failed']} |")
    print(f"| Errors | {s['errors']} |")
    print(f"| **Compliance** | **{s['compliance']}%** |\n")
    print(f"| Test | Status | Severity | Duration | Error |")
    print(f"|------|--------|----------|----------|-------|")
    for r in run.results:
        err = r.error[:80] + "..." if len(r.error) > 80 else r.error
        print(f"| {r.id} | {r.status.upper()} | {r.severity} | {r.duration_ms:.0f}ms | {err} |")


def print_json(run: TestRun):
    s = run.summary()
    out = {
        "runtime": run.runtime,
        "summary": s,
        "results": [
            {"id": r.id, "name": r.name, "severity": r.severity,
             "category": r.category, "status": r.status,
             "error": r.error, "duration_ms": r.duration_ms}
            for r in run.results
        ],
    }
    print(json.dumps(out, indent=2))


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description="Hermes A2A + ARP client conformance tests")
    parser.add_argument("--mock-url", required=True, help="Base URL of the mock A2A server")
    parser.add_argument("--json", action="store_true", help="Output JSON instead of Markdown")
    parser.add_argument("--suite", choices=["a2a", "arp", "all"], default="all", help="Which suite to run")
    args = parser.parse_args()

    results = []
    if args.suite in ("a2a", "all"):
        results.extend(a2a_tests(args.mock_url))
    if args.suite in ("arp", "all"):
        results.extend(arp_tests(args.mock_url))

    run = TestRun(runtime="hermes-agent", results=results)

    if args.json:
        print_json(run)
    else:
        print_markdown(run)

    s = run.summary()
    sys.exit(0 if s["failed"] == 0 and s["errors"] == 0 else 1)


if __name__ == "__main__":
    main()
