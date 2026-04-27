# A2A Protocol Conformance Test Reports

## Test Agents

Two A2A-compatible agents were tested:

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

## Results Summary

| Agent | Compliance | Passed | Failed | Errors |
|-------|-----------|--------|--------|--------|
| a2a-js-sample | **94.7%** | 18 | 1 | 0 |
| a2a-python-helloworld | **47.4%** | 9 | 10 | 0 |

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

## Files

- `a2a-js-sample.md` / `.json` — Full results for the JS sample agent
- `a2a-python-helloworld.md` / `.json` — Full results for the Python helloworld agent
