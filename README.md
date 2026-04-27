# spec-torture

Torture-test agent runtimes against arbitrary protocol specifications.

## Overview

spec-torture is a conformance test harness for AI agent protocols. It validates whether agent runtimes correctly implement protocol specifications by defining test suites in YAML and executing them against live endpoints or Docker containers.

**Supported transports:**
- **HTTP REST** — direct HTTP requests (GET, POST) with JSON body matching
- **JSON-RPC over HTTP** — JSON-RPC 2.0 request/response over HTTP POST
- **JSON-RPC over stdio** — JSON-RPC 2.0 over stdin/stdout (Docker containers)
- **SSE streaming** — Server-Sent Events for streaming protocols
- **NDJSON streaming** — newline-delimited JSON event streams

## Test Suites

| Spec | Version | Transport | Tests | Description |
|------|---------|-----------|-------|-------------|
| [A2A](specs/a2a/) | v1.0 | `jsonrpc-http` | 19 | [Agent-to-Agent Protocol](https://a2a-protocol.org/latest/) — Google's open protocol for agent interop. Tests discovery (`.well-known/agent-card.json`), task lifecycle, messaging (text/file/data parts), SSE streaming, and JSON-RPC error handling. |
| [ACP](specs/acp/) | v0.1 | `http-rest` | 34 | [Crush Agent Communication Protocol](https://github.com/charmbracelet/crush) — HTTP REST protocol for agent runs. Tests health, agent discovery, sync/async/streaming run modes, NDJSON event streams, session management, input validation, and cancellation. |
| [MCP](specs/mcp/) | v0.1 | `jsonrpc-stdio` | 3 | [Model Context Protocol](https://modelcontextprotocol.io/) — skeleton spec for stdio-based tool servers. |

## Test Results

### A2A v1.0 — Agent-to-Agent Protocol

Tested against three A2A runtimes — two official SDK sample agents and crush-a2a (Crush native protocol):

| Runtime | SDK/Backend | Compliance | Passed | Failed | Notes |
|---------|------------|-----------|--------|--------|-------|
| [crush-a2a](reports/a2a/crush-a2a.md) | Crush native | **100.0%** | 19/19 | 0 | Full v1.0 compliance. All discovery, lifecycle, messaging, streaming, error handling, and push notification tests pass. |
| [a2a-python-helloworld](reports/a2a/a2a-python-helloworld.md) | a2a-sdk v1.0.1 | **42.1%** | 8/19 | 11 | Discovery and JSON-RPC error handling pass. All method-specific tests fail — SDK only recognizes v0.3 method names despite being v1.0 release. |
| [a2a-js-sample](reports/a2a/a2a-js-sample.md) | @a2a-js/sdk v0.3.13 | **42.1%** | 8/19 | 11 | Same failure pattern as Python. Discovery passes, but `SendMessage`/`GetTask` return method-not-found. |

**Key finding — protocol version split:** The A2A ecosystem has a critical v0.3 vs v1.0 incompatibility. The spec suite uses v1.0 PascalCase method names (`SendMessage`, `GetTask`) as published in the [official A2A v1.0 specification](https://a2a-protocol.org/latest/). Both official SDKs (Python v1.0.1 and JS v0.3.13) only recognize the older v0.3 slash-separated names (`message/send`, `tasks/get`). crush-a2a is the only runtime that implements the v1.0 spec as published. Discovery (`.well-known/agent-card.json`) is stable across all versions.

Full analysis: [reports/a2a/README.md](reports/a2a/README.md)

### ACP v0.1 — Crush Agent Communication Protocol

Tested against a local Crush ACP server:

| Runtime | Compliance | Passed | Failed | Notes |
|---------|-----------|--------|--------|-------|
| [crush-local](reports/acp/crush-local.md) | **88.2%** | 30/34 | 4 | Strong compliance. Failures are edge cases in async output population, NDJSON event formatting, and HTTP status code for cancel (202 vs 200). |

Failure details:
- `run-completed-has-output` — async run completes with empty output array (race condition in output population)
- `stream-has-message-parts` — NDJSON `message.part` event structure doesn't match expected schema
- `cancel-active-run` — returns HTTP 202 instead of expected 200
- `session-message-events` — `session.message` event structure differs from expected format

## Quick Start

```bash
# Build
make build

# Validate a spec
./bin/spec-torture validate specs/a2a/spec.yaml

# Run against a live endpoint (HTTP REST or JSON-RPC)
./bin/spec-torture run specs/acp/spec.yaml \
  --runtime crush-local \
  --url http://localhost:9106

# Run against a Docker container (stdio transport)
./bin/spec-torture run specs/mcp/spec.yaml \
  --runtime my-mcp-server \
  --image my-mcp-server:latest

# Output as JSON instead of markdown
./bin/spec-torture run specs/a2a/spec.yaml \
  --runtime my-agent \
  --url http://localhost:8080 \
  --format json

# Filter by tags
./bin/spec-torture run specs/a2a/spec.yaml \
  --runtime my-agent \
  --url http://localhost:8080 \
  --tags streaming

# View a stored test run
./bin/spec-torture report <run-id>
./bin/spec-torture report <run-id> --format json
```

## Writing a Spec

Specs are YAML files defining test cases with send/expect steps:

```yaml
id: my-protocol-v1
name: My Protocol
version: 1.0.0
description: Conformance tests for My Protocol
source_url: https://example.com/spec
transport: http-rest  # or jsonrpc-http, jsonrpc-stdio

test_cases:
  - id: health-check
    name: Health Check
    description: Verify the health endpoint responds
    severity: required
    category: health
    timeout: 5s
    tags: [health, smoke]
    steps:
      - action: send
        payload:
          http_method: GET
          path: "/health"
      - action: expect
        expect:
          http_status: 200
          body:
            status: "ok"
```

### Severity Levels (RFC 2119)

| Severity | Meaning | Compliance Impact |
|----------|---------|-------------------|
| `required` | MUST implement | Failure = non-compliant |
| `recommended` | SHOULD implement | Failure = warning |
| `optional` | MAY implement | Informational only |

### Step Actions

| Action | Description |
|--------|-------------|
| `send` | Send an HTTP request or JSON-RPC message to the runtime |
| `expect` | Match the response against an expected pattern (supports `"*"` wildcards and partial matching) |
| `wait` | Pause for the step's timeout duration |
| `assert` | Check a condition against accumulated state |
| `exec` | Run a command inside the container |

### Multi-Step Sequences

Steps execute in order. The runner stores the last response, so later steps can reference earlier results using Go template variables:

```yaml
steps:
  - action: send
    payload:
      http_method: POST
      path: "/runs"
      body:
        agent_name: "my-agent"
        input: [{ role: "user", parts: [{ content_type: "text/plain", content: "hello" }] }]
        mode: "async"
  - action: expect
    expect:
      http_status: 202
      body:
        status: "created"
  - action: send
    payload:
      http_method: GET
      path: "/runs/{{.last_response.run_id}}"
  - action: expect
    expect:
      http_status: 200
```

## Architecture

```
cmd/spec-torture/          CLI (Cobra) — run, list, report, validate
internal/
  schema/                  Spec, TestCase, Step, TestResult, TestRun, Summary
  runner/
    runner.go              Test execution engine, transport routing
    http_runner.go         HTTP REST transport (ACP, generic REST APIs)
    jsonrpc_runner.go      JSON-RPC 2.0 over HTTP transport (A2A, MCP-over-HTTP)
    evaluator.go           Partial matching with wildcard support
    docker.go              Docker container lifecycle
  report/                  JSON and Markdown report generation
  store/                   SQLite persistence for specs and results
specs/                     Test suite definitions (YAML)
reports/                   Committed test run reports
```

## Requirements

- Go 1.24+
- Docker (only for stdio-transport specs that run runtimes in containers)

## License

MIT
