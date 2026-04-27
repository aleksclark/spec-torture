# A2A Protocol Conformance Test Reports

## Test Agents

Three A2A-compatible agents were tested against the A2A v1.0 spec suite (19 tests):

### 1. crush-a2a (Crush native protocol + A2A v1.0 frontend)
- **Source:** crush-a2a Go server (translates A2A v1.0 JSON-RPC → Crush's native unix socket protocol)
- **Runtime:** Go, single-binary server backed by Crush native server
- **Protocol:** JSON-RPC 2.0 over HTTP, A2A v1.0 method names (`SendMessage`, `GetTask`, `CancelTask`, `SendStreamingMessage`)
- **Port:** localhost:8200

### 2. a2a-python-helloworld (A2A Python SDK v1.0.1)
- **Source:** [a2aproject/a2a-samples](https://github.com/a2aproject/a2a-samples) `samples/python/agents/helloworld`
- **Runtime:** Python + Starlette/Uvicorn, using `a2a-sdk==1.0.1`
- **Protocol:** JSON-RPC 2.0 over HTTP. Agent card reports `protocolVersion: "0.3"` despite using v1.0 SDK. Method routing uses v0.3 style internally.
- **Port:** localhost:9999

### 3. a2a-js-sample (A2A-JS SDK v0.3.13)
- **Source:** [a2aproject/a2a-js](https://github.com/a2aproject/a2a-js) `src/samples/agents/sample-agent`
- **Runtime:** Node.js + Express, using `@a2a-js/sdk@0.3.13`
- **Protocol:** JSON-RPC 2.0 over HTTP, A2A v0.3 method names (`message/send`, `tasks/get`, `message/stream`)
- **Port:** localhost:41241

## Results Summary

| Agent | SDK Version | Compliance | Passed | Failed | Errors |
|-------|------------|-----------|--------|--------|--------|
| crush-a2a | Crush native | **100.0%** | 19/19 | 0 | 0 |
| a2a-python-helloworld | a2a-sdk v1.0.1 | **42.1%** | 8/19 | 11 | 0 |
| a2a-js-sample | @a2a-js/sdk v0.3.13 | **42.1%** | 8/19 | 11 | 0 |

## Detailed Results

### crush-a2a — 100% Compliance (19/19)

All 19 tests pass:
- **Discovery (4/4):** Agent card served correctly with all required and optional fields
- **Lifecycle (5/5):** SendMessage, GetTask, GetTask-not-found, CancelTask, and context test all pass
- **Messaging (3/3):** Text parts, context sharing, and optional messageId all work correctly
- **Streaming (2/2):** SSE stream with proper content-type and task status events
- **Error handling (4/4):** Parse error (-32700), invalid request (-32600), method not found (-32601), invalid params (-32602)
- **Push notifications (1/1):** Correctly returns error when unsupported

### a2a-python-helloworld — 42.1% Compliance (8/19)

**Passes (8):** All 4 discovery tests, get-task-not-found, parse-error, invalid-request, method-not-found

**Fails (11):**
- All `SendMessage`-based tests fail with `missing key at result` — the Python SDK v1.0 returns JSON-RPC errors instead of results for v1.0 method names like `SendMessage`. Despite using `a2a-sdk==1.0.1`, the agent card reports `protocolVersion: "0.3"` and the SDK's internal routing doesn't map v1.0 PascalCase methods correctly.
- `streaming-send` / `streaming-events` — returns `application/json` instead of `text/event-stream`
- `missing-required-params` — returns `-32009` instead of `-32602` (custom error code vs standard invalid params)

### a2a-js-sample — 42.1% Compliance (8/19)

**Passes (8):** All 4 discovery tests, get-task-not-found, parse-error, invalid-request, method-not-found

**Fails (11):**
- Identical failure pattern to Python: all `SendMessage`-based tests fail because the JS SDK only recognizes v0.3 method names (`message/send`, `tasks/get`), not v1.0 names (`SendMessage`, `GetTask`).
- `streaming-send` — returns `application/json; charset=utf-8` instead of `text/event-stream`
- `missing-required-params` — returns `-32601` (method not found) instead of `-32602` (invalid params), because it doesn't even recognize `SendMessage` as a valid method

## Key Findings

### Protocol Version Split (v0.3 vs v1.0)

The A2A ecosystem has a critical v0.3 vs v1.0 incompatibility. Our spec suite uses **A2A v1.0 method names** as defined in the [official specification](https://a2a-protocol.org/latest/):

| Operation | v0.3 (JS SDK) | v1.0 (Spec) |
|-----------|---------------|-------------|
| Send message | `message/send` | `SendMessage` |
| Get task | `tasks/get` | `GetTask` |
| Cancel task | `tasks/cancel` | `CancelTask` |
| Stream message | `message/stream` | `SendStreamingMessage` |

**Both the Python SDK v1.0.1 and JS SDK v0.3.13 only recognize v0.3 method names.** The "v1.0" Python SDK appears to be a v1.0 release of the SDK package, not an implementation of the v1.0 protocol spec. Neither SDK supports the PascalCase method names from the published v1.0 specification.

This means:
- **crush-a2a is the only runtime that implements the v1.0 spec as published**
- Both official SDK sample agents implement v0.3 protocol only
- The v0.3→v1.0 method name migration has not been adopted by the official SDKs

### What Passes Across All Three

- **Discovery** (`.well-known/agent-card.json`): All agents serve valid agent cards via GET
- **JSON-RPC error handling**: All correctly return -32700 (parse error), -32600 (invalid request), and -32601 (method not found)
- **Content-Type**: All return `application/json` for agent cards

### Error Code Differences

For `missing-required-params` (sending `SendMessage` with empty params):
- crush-a2a: `-32602` (correct — invalid params) ✓
- Python SDK: `-32009` (custom A2A error code — not standard JSON-RPC)
- JS SDK: `-32601` (method not found — doesn't recognize `SendMessage` at all)

## How to Reproduce

```bash
# 1. Start crush-a2a (assumes it's already running on port 8200)
go run ./cmd/spec-torture run specs/a2a/spec.yaml --runtime crush-a2a --url http://localhost:8200

# 2. Start the Python helloworld agent
cd /tmp && git clone --depth 1 https://github.com/a2aproject/a2a-samples.git
cd a2a-samples/samples/python/agents/helloworld
uv run . &
go run ./cmd/spec-torture run specs/a2a/spec.yaml --runtime a2a-python-helloworld --url http://localhost:9999

# 3. Start the JS sample agent
cd /tmp && git clone --depth 1 https://github.com/a2aproject/a2a-js.git
cd a2a-js && npm install && npm run build
PORT=41241 npx tsx src/samples/agents/sample-agent/index.ts &
go run ./cmd/spec-torture run specs/a2a/spec.yaml --runtime a2a-js-sample --url http://localhost:41241
```

## Files

- `crush-a2a.md` / `.json` — Full results for crush-a2a (19/19, 100%)
- `a2a-python-helloworld.md` / `.json` — Full results for the Python helloworld agent (8/19, 42.1%)
- `a2a-js-sample.md` / `.json` — Full results for the JS sample agent (8/19, 42.1%)
