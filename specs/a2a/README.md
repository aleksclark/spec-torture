# A2A Protocol v1.0 Test Suite

Conformance tests for the [Agent-to-Agent (A2A) protocol](https://google.github.io/A2A) v1.0.

## Overview

This suite validates A2A agent runtimes across 29 test cases covering the full protocol surface: agent discovery, task lifecycle, message handling, streaming, error semantics, and push notifications.

Transport: **JSON-RPC 2.0 over HTTP POST** (`jsonrpc-http`)

## Test Categories

| Category | Tests | Severity Breakdown |
|---|---|---|
| **discovery** | 4 | 2 required, 1 recommended, 1 optional |
| **lifecycle** | 8 | 6 required, 2 recommended |
| **messaging** | 5 | 3 required, 2 recommended |
| **streaming** | 4 | 4 optional |
| **error-handling** | 5 | 5 required |
| **push-notifications** | 3 | 3 optional |

### Discovery

Tests the `/.well-known/agent-card.json` endpoint and the `GetExtendedAgentCard` RPC. Validates that agent metadata (name, version, URL, skills) is present and correctly served.

### Lifecycle

Tests the core task lifecycle: `SendMessage`, `GetTask`, `ListTasks`, and `CancelTask`. Validates task creation, state transitions (submitted → working → completed/canceled), and error conditions for nonexistent or terminal tasks.

### Messaging

Tests message part types (`TextPart`, `FilePart`, `DataPart`), context grouping via `contextId`, and error handling when messaging completed tasks.

### Streaming

Tests `SendStreamingMessage` SSE streams, `TaskStatusUpdateEvent` and `TaskArtifactUpdateEvent` delivery, and `SubscribeToTask` resubscription.

### Error Handling

Tests JSON-RPC 2.0 error codes (`-32700` parse error, `-32600` invalid request, `-32601` method not found), unsupported content types, and authentication enforcement.

### Push Notifications

Tests `CreateTaskPushNotificationConfig`, `GetTaskPushNotificationConfig`, and `DeleteTaskPushNotificationConfig` for webhook-based task event delivery.

## Running

```bash
# Validate the spec
spec-torture validate specs/a2a/spec.yaml

# Run against an A2A agent runtime
spec-torture run specs/a2a/spec.yaml --runtime my-a2a-agent --image my-a2a-agent:latest

# Run only required tests
spec-torture run specs/a2a/spec.yaml --runtime my-a2a-agent --image my-a2a-agent:latest --tags required

# Run only streaming tests
spec-torture run specs/a2a/spec.yaml --runtime my-a2a-agent --image my-a2a-agent:latest --tags streaming
```

## Severity Levels

| Level | Count | Meaning |
|---|---|---|
| `required` | 16 | MUST implement — failure means non-compliant |
| `recommended` | 5 | SHOULD implement — failure is a warning |
| `optional` | 8 | MAY implement — informational only |

## A2A Protocol Reference

- [A2A Specification](https://google.github.io/A2A)
- [Agent Card schema](https://google.github.io/A2A/#/documentation?id=agent-card)
- [JSON-RPC methods](https://google.github.io/A2A/#/documentation?id=methods)
- [Task lifecycle](https://google.github.io/A2A/#/documentation?id=task)
- [Streaming (SSE)](https://google.github.io/A2A/#/documentation?id=streaming)
