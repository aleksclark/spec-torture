# A2A Protocol Conformance Test Reports

## Test Agents

Three A2A-compatible agents were tested:

### 1. a2a-js-sample (A2A-JS SDK v0.3 format)
- **Source:** [a2aproject/a2a-js](https://github.com/a2aproject/a2a-js) `src/samples/agents/sample-agent`
- **Runtime:** Node.js + Express, using the A2A-JS SDK
- **Protocol:** JSON-RPC 2.0 over HTTP, A2A v0.3 method names (`message/send`, `tasks/get`, `message/stream`)
- **Port:** localhost:41241

### 2. a2a-python-helloworld (A2A Python SDK v1.0 format)
- **Source:** [a2aproject/a2a-samples](https://github.com/a2aproject/a2a-samples) `samples/python/agents/helloworld`
- **Runtime:** Python + Starlette/Uvicorn, using `a2a-sdk==1.0.1`
- **Protocol:** JSON-RPC 2.0 over HTTP, A2A v1.0 method names (`SendMessage`, `GetTask`) with protobuf-style JSON
- **Port:** localhost:9999

### 3. crush-a2a-native (Crush AI via native protocol + A2A v1.0 frontend)
- **Source:** crush-a2a Go server (translates A2A v1.0 JSON-RPC → Crush's native unix socket protocol)
- **Runtime:** Go, single-binary server backed by Crush native server
- **Protocol:** JSON-RPC 2.0 over HTTP, A2A v1.0 method names (`SendMessage`, `GetTask`, `CancelTask`, `SendStreamingMessage`)
- **Port:** localhost:8200

## Results Summary

| Agent | Compliance | Passed | Failed | Errors |
|-------|-----------|--------|--------|--------|
| a2a-js-sample | **94.7%** | 18 | 1 | 0 |
| a2a-python-helloworld | **47.4%** | 9 | 10 | 0 |
| crush-a2a-native | **89.5%** | 17 | 2 | 0 |

## Key Findings

### Protocol Version Split (v0.3 vs v1.0)
The A2A ecosystem has a significant protocol version split:

- **v0.3** (a2a-js): Uses slash-separated method names (`message/send`, `tasks/get`, `message/stream`), camelCase field names (`messageId`, `contextId`), and `kind` discriminators in parts.
- **v1.0** (a2a-python): Uses PascalCase method names (`SendMessage`, `GetTask`), protobuf-style enums (`ROLE_USER`, `TASK_STATE_COMPLETED`), snake_case fields (`message_id`), and requires `A2A-Version: 1.0` header.

The Python v1.0 SDK rejects all v0.3 method names with `-32601 Method not found`, meaning agents built with different SDK versions cannot interoperate without an adapter layer.

### What Passes Across Both
- **Discovery** (`.well-known/agent-card.json`): Both agents serve valid agent cards via GET
- **JSON-RPC error handling**: Both correctly return -32700 (parse error), -32600 (invalid request), and -32601 (method not found)
- **Content-Type**: Both return `application/json` for agent cards

### JS Agent (94.7%)
- Passes all discovery, lifecycle, messaging, streaming, and error-handling tests
- Single failure: `missing-required-params` returns `-32603` instead of `-32602` (internal error vs invalid params)
- Streaming (SSE) works correctly

### Python Agent (47.4%)
- Passes discovery and JSON-RPC standard error codes
- Fails all method-specific tests because v1.0 SDK doesn't support v0.3 method names
- The agent is fully functional — just speaks a different protocol dialect

### crush-a2a-native (89.5%)
- **Discovery (4/4 pass):** Agent card served correctly at `/.well-known/agent-card.json` with name, version, url, skills, description, and capabilities
- **Lifecycle (3/4 pass):** `SendMessage` creates tasks with proper id/contextId/status, `CancelTask` works on active tasks. Single failure: `GetTask` returns an error instead of the task result because the proxy is stateless and does not persist completed tasks after returning the final response.
- **Messaging (2/2 pass):** Text parts processed correctly, context sharing via `contextId` works
- **Streaming (2/2 pass):** `SendStreamingMessage` returns SSE stream with `text/event-stream` content type and proper task status/artifact update events
- **Error handling (5/5 pass for standard JSON-RPC, 1/1 fail for A2A validation):** JSON-RPC -32700 (parse error), -32600 (invalid request), -32601 (method not found), and -32602 (invalid params for missing `message` field) all correct. Single remaining failure:
  - `missing-message-id`: Returns a successful result instead of a validation error because the proxy does not enforce the A2A requirement that `messageId` be present in every `Message` object
- **Push notifications (1/1 pass):** Correctly returns error when push notification config is set on an agent that doesn't support it
- **Improvement over previous run (v2):** Compliance rose from 84.2% to 89.5%. The native protocol backend now correctly returns -32602 for missing required params (previously returned -32603). Timeout issues in multi-step tests were resolved by increasing test timeouts to accommodate real LLM response times.
- **Root cause of remaining failures:**
  1. `get-task`: Architectural — the stateless proxy does not maintain a task store, so `GetTask` cannot retrieve previously completed tasks
  2. `missing-message-id`: The proxy does not validate that `messageId` is present in `Message` objects before forwarding to the native backend

## How to Reproduce

```bash
# 1. Start the JS sample agent
cd /tmp && git clone --depth 1 https://github.com/a2aproject/a2a-js.git
cd a2a-js && npm install && npm run build
cd src/samples && npm install
PORT=41241 npm run agents:sample-agent &

# 2. Start the Python helloworld agent
cd /tmp && git clone --depth 1 https://github.com/a2aproject/a2a-samples.git
cd a2a-samples/samples/python/agents/helloworld
uv run . &

# 3. Run the conformance suite
cd spec-torture
go run ./cmd/spec-torture run specs/a2a/spec.yaml --runtime a2a-js-sample --url http://localhost:41241
go run ./cmd/spec-torture run specs/a2a/spec.yaml --runtime a2a-python-helloworld --url http://127.0.0.1:9999
```

## How to Reproduce crush-a2a-native

```bash
# 1. Start crush-a2a backed by Crush's native protocol server
# (assumes crush-a2a binary is running on port 8200, connected to Crush via unix socket)

# 2. Run conformance suite
cd spec-torture
go run ./cmd/spec-torture run specs/a2a/spec.yaml --runtime crush-a2a-native --url http://localhost:8200
go run ./cmd/spec-torture run specs/a2a/spec.yaml --runtime crush-a2a-native --url http://localhost:8200 --format json
```

## Files

- `a2a-js-sample.md` / `.json` — Full results for the JS sample agent
- `a2a-python-helloworld.md` / `.json` — Full results for the Python helloworld agent
- `crush-a2a.md` / `.json` — Full results for crush-a2a (native protocol backend)
