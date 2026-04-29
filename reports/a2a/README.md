# A2A Protocol Conformance Test Reports

## Test Agents

Four A2A-compatible agents tested against the A2A v1.0 spec suite (19 tests):

### 1. a2a-go-helloworld (a2a-go SDK v2.x)
- **Source:** [a2aproject/a2a-go](https://github.com/a2aproject/a2a-go) `examples/helloworld/server/jsonrpc`
- **Runtime:** Go, using `a2a-go/v2` SDK
- **Protocol:** JSON-RPC 2.0 over HTTP at `/invoke`, A2A v1.0 PascalCase method names
- **Port:** localhost:8201

### 2. a2a-rs-helloworld (a2a-rs SDK)
- **Source:** [a2aproject/a2a-rs](https://github.com/a2aproject/a2a-rs) `examples/helloworld`
- **Runtime:** Rust + Axum + Tokio, using `a2a` and `a2a-server` crates
- **Protocol:** JSON-RPC 2.0 over HTTP at `/jsonrpc`, A2A v1.0 PascalCase method names
- **Port:** localhost:8202

### 3. a2a-python-helloworld (A2A Python SDK v1.0.1)
- **Source:** [a2aproject/a2a-samples](https://github.com/a2aproject/a2a-samples) `samples/python/agents/helloworld`
- **Runtime:** Python + Starlette/Uvicorn, using `a2a-sdk==1.0.1`
- **Protocol:** JSON-RPC 2.0 over HTTP. Agent card reports `protocolVersion: "0.3"` despite using v1.0 SDK. Method routing uses v0.3 style internally.
- **Port:** localhost:9999

### 4. a2a-js-sample (A2A-JS SDK v0.3.13)
- **Source:** [a2aproject/a2a-js](https://github.com/a2aproject/a2a-js) `src/samples/agents/sample-agent`
- **Runtime:** Node.js + Express, using `@a2a-js/sdk@0.3.13`
- **Protocol:** JSON-RPC 2.0 over HTTP, A2A v0.3 method names (`message/send`, `tasks/get`, `message/stream`)
- **Port:** localhost:41241

## Results Summary

| Agent | SDK Version | Compliance | Passed | Failed | Errors | Notes |
|-------|------------|-----------|--------|--------|--------|-------|
| a2a-go-helloworld | a2a-go v2.x | **47.4%** | 9/19 | 10 | 0 | Good v1.0 method support; response envelope differs from spec expectations |
| a2a-rs-helloworld | a2a-rs (Rust) | **26.3%** | 5/19 | 14 | 0 | Rejects `kind` field in parts; HTTP errors for malformed JSON instead of JSON-RPC errors |
| a2a-python-helloworld | a2a-sdk v1.0.1 | **42.1%** | 8/19 | 11 | 0 | Discovery and JSON-RPC error handling pass; v0.3 method names only |
| a2a-js-sample | @a2a-js/sdk v0.3.13 | **42.1%** | 8/19 | 11 | 0 | Same v0.3-only failure pattern as Python |

## Detailed Results

### a2a-go-helloworld — 47.4% Compliance (9/19)

**Passes (9):** agent-card-content-type, agent-card-optional-fields, agent-card-skills, get-task-not-found, streaming-send (SSE content-type correct), and all 4 JSON-RPC error handling tests.

**Fails (10):**

| Test | Error | Analysis |
|------|-------|----------|
| agent-card-well-known | `missing key at url` | Agent card uses `supportedInterfaces[].url` per v1.0 spec, but test checks for top-level `url` field |
| send-message-basic | `missing key at result.id` | Response is `result.message{...}` — no Task wrapper with `id`/`contextId` |
| send-message-creates-task | `missing key at result.kind` | Same: returns Message directly, not a Task object |
| get-task | `missing key at result.id` | GetTask depends on successful SendMessage first (cascading failure) |
| cancel-task | `missing key at result.id` | Same cascading dependency issue |
| message-text-part | `missing key at result.id` | Response lacks Task wrapper |
| message-with-context | `missing key at result.id` | Response lacks Task wrapper |
| streaming-events | `no SSE event matched: missing key at taskId` | SSE events don't contain `taskId` at the expected path |
| optional-message-id | `missing key at result` | Message without `messageId` returns different response structure |
| push-notification-not-supported | `missing key at result.id` | Response structure differs |

**Key finding:** The Go SDK v2 returns `result.message` directly from `SendMessage` instead of wrapping it in a `Task` object with `id`, `contextId`, and `status`. The spec expects the full Task lifecycle model. The Go SDK also uses `ROLE_AGENT` enum strings instead of the expected `"agent"` string for the role field.

### a2a-rs-helloworld — 26.3% Compliance (5/19)

**Passes (5):** agent-card-content-type, agent-card-optional-fields, agent-card-skills, get-task-not-found, jsonrpc-method-not-found.

**Fails (14):**

| Test | Error | Analysis |
|------|-------|----------|
| agent-card-well-known | `missing key at url` | Same as Go: uses `supportedInterfaces` not top-level `url` |
| send-message-basic | `missing key at result` | Returns error: `unknown field 'kind'` — rejects `kind` in message parts |
| send-message-creates-task | `missing key at result` | Same `kind` field rejection |
| get-task | `missing key at result` | Cascading from SendMessage failure |
| cancel-task | `missing key at result` | Same cascading failure |
| message-text-part | `missing key at result` | Same `kind` field rejection |
| message-with-context | `missing key at result` | Same `kind` field rejection |
| streaming-send | `content-type: expected "text/event-stream", got "application/json"` | Same `kind` rejection prevents streaming |
| streaming-events | `expected SSE events but got none` | Same root cause |
| jsonrpc-parse-error | `expected JSON-RPC response but got none` | Returns HTTP 400 with plaintext instead of JSON-RPC error |
| jsonrpc-invalid-request | `expected JSON-RPC response but got none` | Returns HTTP 422 with plaintext instead of JSON-RPC error |
| missing-required-params | `missing key at error` | Accepts empty params and returns success instead of error |
| optional-message-id | `missing key at result` | Same `kind` rejection |
| push-notification-not-supported | `missing key at result` | Same `kind` rejection |

**Key findings:**
1. **Part field naming:** The Rust SDK uses protobuf-style field names and rejects `kind` — expects `text`, `raw`, `url`, `data`, etc. as direct fields. The spec's test payloads use `kind: "text"` + `text: "..."` which is the v1.0 JSON representation.
2. **HTTP errors for JSON-RPC:** Malformed JSON returns HTTP 400/422 with plaintext body instead of a JSON-RPC `-32700` error response. The JSON-RPC 2.0 spec requires errors to be returned as JSON-RPC error objects.
3. **Missing params handling:** The Rust SDK accepts `SendMessage` with empty params and processes it (returns a task echoing `(no message)`) instead of returning `-32602 Invalid params`.
4. **Response wrapping:** When it does succeed, the Rust SDK wraps responses as `result.task{...}` instead of `result{id, contextId, status, ...}`.

### a2a-python-helloworld — 42.1% Compliance (8/19)

**Passes (8):** All 4 discovery tests, get-task-not-found, parse-error, invalid-request, method-not-found

**Fails (11):**
- All `SendMessage`-based tests fail with `missing key at result` — the Python SDK v1.0 returns JSON-RPC errors instead of results for v1.0 method names like `SendMessage`. Despite using `a2a-sdk==1.0.1`, the agent card reports `protocolVersion: "0.3"` and the SDK's internal routing doesn't map v1.0 PascalCase methods correctly.
- `streaming-send` / `streaming-events` — returns `application/json` instead of `text/event-stream`
- `missing-required-params` — returns `-32009` instead of `-32602` (custom error code vs standard JSON-RPC)

### a2a-js-sample — 42.1% Compliance (8/19)

**Passes (8):** All 4 discovery tests, get-task-not-found, parse-error, invalid-request, method-not-found

**Fails (11):**
- Identical failure pattern to Python: all `SendMessage`-based tests fail because the JS SDK only recognizes v0.3 method names (`message/send`, `tasks/get`), not v1.0 names (`SendMessage`, `GetTask`).
- `streaming-send` — returns `application/json; charset=utf-8` instead of `text/event-stream`
- `missing-required-params` — returns `-32601` (method not found) instead of `-32602` (invalid params), because it doesn't even recognize `SendMessage` as a valid method

## Cross-Runtime Analysis

### Protocol Version Adoption

| Feature | Go SDK v2 | Rust SDK | Python SDK v1.0.1 | JS SDK v0.3.13 |
|---------|-----------|----------|-------------------|----------------|
| v1.0 PascalCase methods | ✅ | ✅ | ❌ (v0.3 only) | ❌ (v0.3 only) |
| `supportedInterfaces` in agent card | ✅ (no `url`) | ✅ (no `url`) | ✅ (+ `url`) | ✅ (+ `url`) |
| Task wrapper in response | ❌ (Message only) | ✅ (as `result.task`) | N/A | N/A |
| JSON-RPC error for malformed input | ✅ | ❌ (HTTP errors) | ✅ | ✅ |
| SSE streaming support | ✅ | ❌ (field error) | ❌ | ❌ |
| `kind` field in message parts | ✅ | ❌ (rejects) | N/A | N/A |

### What Passes Across All Four Runtimes

- **Agent card content-type and optional fields** — all serve valid agent cards via GET at `/.well-known/agent-card.json`
- **Agent card skills** — all include at least one skill with `id` and `name`
- **get-task-not-found** — all return proper JSON-RPC error for unknown task IDs

### Response Envelope Differences (v1.0 SDKs)

The two v1.0-capable runtimes return different response structures for `SendMessage`:

```
a2a-go v2:   result: { message: { messageId, parts, role: "ROLE_AGENT" } }   (Message only)
a2a-rs:      result: { task: { id, contextId, status: { message: { ... } } } } (Task in wrapper)
```

The two don't agree on the response structure, highlighting that the v1.0 specification's response format is interpreted differently by each SDK.

### Error Handling Spectrum

For malformed JSON input (`{invalid json`):
- Go SDK: `-32700` JSON-RPC error ✅
- Rust SDK: HTTP 400 plaintext ❌
- Python SDK: `-32700` JSON-RPC error ✅
- JS SDK: `-32700` JSON-RPC error ✅

## How to Reproduce

```bash
# 1. a2a-go helloworld (JSON-RPC at /invoke)
cd /tmp && git clone --depth 1 https://github.com/a2aproject/a2a-go.git
cd a2a-go && go run ./examples/helloworld/server/jsonrpc --port 8201 &
go run ./cmd/spec-torture run specs/a2a/spec.yaml \
  --runtime a2a-go-helloworld --url http://localhost:8201 --rpc-path /invoke

# 2. a2a-rs helloworld (JSON-RPC at /jsonrpc, default port 3000)
cd /tmp && git clone --depth 1 https://github.com/a2aproject/a2a-rs.git
cd a2a-rs && cargo run -p helloworld &
go run ./cmd/spec-torture run specs/a2a/spec.yaml \
  --runtime a2a-rs-helloworld --url http://localhost:3000 --rpc-path /jsonrpc

# 3. Python helloworld
cd /tmp && git clone --depth 1 https://github.com/a2aproject/a2a-samples.git
cd a2a-samples/samples/python/agents/helloworld && uv run . &
go run ./cmd/spec-torture run specs/a2a/spec.yaml \
  --runtime a2a-python-helloworld --url http://localhost:9999

# 4. JS sample agent
cd /tmp && git clone --depth 1 https://github.com/a2aproject/a2a-js.git
cd a2a-js && npm install && npm run build
PORT=41241 npx tsx src/samples/agents/sample-agent/index.ts &
go run ./cmd/spec-torture run specs/a2a/spec.yaml \
  --runtime a2a-js-sample --url http://localhost:41241
```

## Files

- `a2a-go-helloworld.md` / `.json` — Go SDK v2 helloworld results (9/19, 47.4%)
- `a2a-rs-helloworld.md` / `.json` — Rust SDK helloworld results (5/19, 26.3%)
- `a2a-python-helloworld.md` / `.json` — Python SDK v1.0.1 helloworld results (8/19, 42.1%)
- `a2a-js-sample.md` / `.json` — JS SDK v0.3.13 sample agent results (8/19, 42.1%)
