# ACP (Agent Communication Protocol) Test Suite

Conformance tests for the Crush Agent Communication Protocol over HTTP REST.

## What This Tests

The suite validates 34 test cases across 8 categories:

| Category | Tests | Description |
|---|---|---|
| **health** | 2 | `GET /ping` liveness and content-type |
| **discovery** | 4 | Agent listing, manifest fields, lookup, 404 handling |
| **runs-sync** | 4 | Synchronous run creation, required fields, output structure |
| **runs-async** | 4 | Async run creation, polling, completion, output |
| **runs-stream** | 6 | NDJSON streaming, event types, headers, content-type |
| **validation** | 4 | Input rejection: empty input, bad agent, empty content, malformed JSON |
| **cancellation** | 3 | Cancel active/completed/nonexistent runs |
| **sessions** | 3 | Session persistence, export, import |
| **events** | 4 | Event listing, NDJSON streaming, session.message and session.snapshot events |

## Severity Breakdown

- **Required** (30 tests): Core protocol compliance — failure means non-conformant
- **Recommended** (4 tests): Expected behavior — failure is a warning

## Running Against a Crush Agent

### Validate the spec

```bash
spec-torture validate specs/acp/spec.yaml
```

### Run the full suite

```bash
spec-torture run specs/acp/spec.yaml \
  --runtime crush \
  --image crush-acp:latest
```

### Filter by tag

```bash
# Only streaming tests
spec-torture run specs/acp/spec.yaml \
  --runtime crush \
  --image crush-acp:latest \
  --tags streaming

# Only session tests
spec-torture run specs/acp/spec.yaml \
  --runtime crush \
  --image crush-acp:latest \
  --tags sessions
```

## ACP Protocol Summary

The Agent Communication Protocol exposes agents over HTTP REST:

- **Health**: `GET /ping` → `"pong"`
- **Discovery**: `GET /agents`, `GET /agents/{name}`
- **Runs**: `POST /runs` with modes `sync`, `async`, `stream`
- **Polling**: `GET /runs/{run_id}`
- **Events**: `GET /runs/{run_id}/events` (JSON array or NDJSON stream)
- **Cancellation**: `POST /runs/{run_id}/cancel`
- **Sessions**: `GET /sessions/{id}/export`, `POST /sessions/import`

Messages use the format `{role, parts: [{content_type, content}]}`.
NDJSON events include `run.created`, `run.in-progress`, `run.completed`, `message.part`, `session.message`, `session.snapshot`, and others.
