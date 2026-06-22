---
title: "ARP — Agent Lifecycle Service"
version: 0.5.0
created: 2026-04-06
updated: 2026-05-01
status: draft
tags: [arp, grpc, protobuf, a2a, agent-lifecycle]
---

# Agent Lifecycle Service

gRPC service for spawning, monitoring, and stopping A2A agents within workspaces. This is the core of ARP's control plane: it manages agent **processes** and hands clients the agent's A2A endpoint.

Agents are the bottom of the hierarchy: **Project → Workspace → Agent**. Each agent is an independent A2A-speaking process with its own port and `AgentCard`. Multiple agents can share one workspace directory.

> **Scope of this service.** `AgentService` covers lifecycle only: spawn, list, get status, stop, restart. It deliberately contains **no messaging RPCs**. To send a message, create a task, or poll task status, talk A2A directly to the agent's `direct_url` — see [Talking to an agent](#talking-to-an-agent-a2a-not-arp) below. This is the ARP/A2A boundary: **ARP manages the agent; A2A talks to it.**

## Service Definition

```protobuf
syntax = "proto3";

package arp.v1;

import "google/api/annotations.proto";
import "google/api/field_behavior.proto";
import "arp/v1/types.proto";
import "arp/v1/workspace.proto";

// AgentService manages the lifecycle of A2A agent instances within workspaces.
// It does not relay agent messages — clients use the agent's direct_url with
// any A2A client for that.
service AgentService {
  // SpawnAgent spawns a new A2A agent instance in a workspace.
  rpc SpawnAgent(SpawnAgentRequest) returns (AgentInstance) {
    option (google.api.http) = {
      post: "/v1/agents"
      body: "*"
    };
  }

  // ListAgents lists all agent instances, optionally filtered.
  rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse) {
    option (google.api.http) = {
      get: "/v1/agents"
    };
  }

  // GetAgentStatus returns detailed status of a specific agent instance,
  // including its resolved AgentCard and direct_url.
  rpc GetAgentStatus(GetAgentStatusRequest) returns (AgentInstance) {
    option (google.api.http) = {
      get: "/v1/agents/{agent_id}"
    };
  }

  // StopAgent gracefully stops an agent instance.
  rpc StopAgent(StopAgentRequest) returns (AgentInstance) {
    option (google.api.http) = {
      post: "/v1/agents/{agent_id}:stop"
      body: "*"
    };
  }

  // RestartAgent restarts an agent instance (stop + spawn with same config).
  rpc RestartAgent(RestartAgentRequest) returns (AgentInstance) {
    option (google.api.http) = {
      post: "/v1/agents/{agent_id}:restart"
      body: "*"
    };
  }
}
```

## Messages

### SpawnAgent

```protobuf
message SpawnAgentRequest {
  // Workspace to spawn agent in.
  string workspace = 1 [(google.api.field_behavior) = REQUIRED];

  // Agent template name (e.g., "crush", "claude-code", "reviewer").
  string template = 2 [(google.api.field_behavior) = REQUIRED];

  // Instance name override (default: template name).
  // Required when spawning multiple agents of the same template.
  string name = 3;

  // Additional environment variables for this instance.
  map<string, string> env = 4;

  // Narrow the spawned agent's project scope (must be subset of caller's scope).
  // Omit to inherit caller's full scope. See identity-and-scopes.md.
  Scope scope = 5;

  // Permission level for spawned agent (must be ≤ caller's permission).
  // Omit to inherit caller's permission. See identity-and-scopes.md.
  Permission permission = 6;
}
```

**Returns:** `AgentInstance` with `direct_url` set once the agent is ready.

**Behavior:**
1. Validates the caller's token — checks scope includes the workspace's project, and permission allows spawning.
2. Allocates a port from the configured range (9100–9199).
3. Issues a child token to the new agent (scope ≤ caller's scope, permission ≤ caller's permission).
4. Starts the agent process with the template's `command`, setting `port_env` to the allocated port and `ARP_TOKEN` to the child token.
5. Polls the `health_check.path` endpoint until it responds (or retries exhausted → status `AGENT_STATUS_ERROR`).
6. On health-check pass, status transitions to `AGENT_STATUS_READY` and `direct_url` is populated.
7. Fetches the agent's `AgentCard` (one A2A discovery `GET`) and enriches it with `metadata.arp`.

> **No initial prompt.** `SpawnAgent` does not send a first message — that would put ARP in the message path. To bootstrap an agent with a task, spawn it, then send an A2A `SendMessage` to its `direct_url` (often the very next call). This keeps spawn a pure lifecycle operation.

**Example:**

```bash
grpcurl -d '{
  "workspace": "feat-auth",
  "template": "crush",
  "name": "coder"
}' localhost:9099 arp.v1.AgentService/SpawnAgent
# → { "id": "coder-abc123", "status": "AGENT_STATUS_READY",
#     "direct_url": "http://localhost:9100", ... }
```

### ListAgents

```protobuf
message ListAgentsRequest {
  // Filter by workspace name.
  string workspace = 1;

  // Filter by ARP agent status.
  AgentStatus status = 2;

  // Filter by template name.
  string template = 3;
}

message ListAgentsResponse {
  repeated AgentInstance agents = 1;
}
```

**Returns:** Array of `AgentInstance` messages (see [Data Model](overview.md#agentinstance)). Visibility is scoped per the caller's token (see [Identity & Scopes](identity-and-scopes.md)).

### GetAgentStatus

```protobuf
message GetAgentStatusRequest {
  // Agent instance ID.
  string agent_id = 1 [(google.api.field_behavior) = REQUIRED];
}
```

**Returns:** `AgentInstance` with `direct_url`, current `status`, and the resolved `AgentCard` (enriched with `metadata.arp`). Use this (or `WatchAgent`) to re-read `direct_url` after a restart.

### StopAgent

Gracefully stop an agent instance.

```protobuf
message StopAgentRequest {
  // Agent instance ID.
  string agent_id = 1 [(google.api.field_behavior) = REQUIRED];

  // Grace period in milliseconds before force kill (default: 5000).
  int32 grace_period_ms = 2;
}
```

**Returns:** `AgentInstance` with final status (`AGENT_STATUS_STOPPED`).

**Behavior:**
1. Transitions agent status to `AGENT_STATUS_STOPPING`.
2. Sends SIGTERM to the agent process.
3. Waits `grace_period_ms` for process exit.
4. If still running, sends SIGKILL.
5. Transitions agent status to `AGENT_STATUS_STOPPED` and frees the allocated port.

> ARP does not cancel in-flight A2A tasks on stop — task cancellation is an A2A concern (`CancelTask` at the agent). If you need a clean drain, send `CancelTask` to the agent's `direct_url` before calling `StopAgent`.

### RestartAgent

Restart an agent instance (stop + spawn with same configuration).

```protobuf
message RestartAgentRequest {
  // Agent instance ID.
  string agent_id = 1 [(google.api.field_behavior) = REQUIRED];
}
```

**Returns:** `AgentInstance` with new status after restart.

**Behavior:**
1. Calls `StopAgent` on the instance.
2. Re-spawns with the same template, name, workspace, and env.
3. A new port may be allocated, so `direct_url` may change — re-read it from the response (the `proxy_url`, when a gateway is deployed, stays stable).

## Talking to an Agent (A2A, not ARP)

Once `SpawnAgent`/`GetAgentStatus` gives you a `direct_url`, you communicate with the agent using **A2A v1.0 directly** — ARP is no longer involved. Use any A2A client; the agent serves the standard A2A HTTP+JSON bindings:

```bash
# 1. Spawn via ARP (gRPC control plane)
DIRECT=$(grpcurl -d '{"workspace":"feat-auth","template":"crush","name":"coder"}' \
  localhost:9099 arp.v1.AgentService/SpawnAgent | jq -r .directUrl)

# 2. Send a message via A2A — straight to the agent, no ARP hop
curl -X POST "$DIRECT/message:send" \
  -H 'Content-Type: application/json' \
  -d '{"message":{"role":"ROLE_USER","parts":[{"text_part":{"text":"Implement OAuth2 login"}}]}}'
# → SendMessageResponse: { "task": { "id": "task-1", "status": { "state": "TASK_STATE_WORKING" } } }

# 3. Poll the task via A2A
curl "$DIRECT/tasks/task-1"
# → Task: { "id": "task-1", "status": { "state": "TASK_STATE_COMPLETED" }, "artifacts": [...] }

# 4. Stop via ARP when done (gRPC control plane)
grpcurl -d '{"agentId":"coder-abc123"}' localhost:9099 arp.v1.AgentService/StopAgent
```

Why this split:

- **Smaller, clearer protocol.** ARP doesn't re-export A2A's message/task surface or wrap its `SendMessageResponse` oneof. One way to send a message: A2A, at the agent.
- **No protocol drift.** A2A messaging features (streaming, push, new part types) are available immediately without ARP changes.
- **Lowest latency.** No proxy hop on the hot path.

If you need a single authenticated A2A entry point (private networks, central auth, skill-based routing), deploy the optional **[A2A gateway](profile-a2a-gateway.md)** and use the agent's `proxy_url`; it forwards transparently to `direct_url` after enforcing token scope.

## gRPC Status Codes

| Condition | gRPC Status | Description |
|-----------|-------------|-------------|
| Agent not found | `NOT_FOUND` | Any RPC with unknown `agent_id` |
| Workspace not found | `NOT_FOUND` | SpawnAgent referencing unknown workspace |
| Template not found | `NOT_FOUND` | SpawnAgent referencing a template not in the project |
| Missing required field | `INVALID_ARGUMENT` | `workspace` or `template` not provided |
| Agent already exists | `ALREADY_EXISTS` | SpawnAgent with a duplicate instance id/name |
| Spawn/health failure | `INTERNAL` | Process failed to start or never became healthy |
| Permission denied | `PERMISSION_DENIED` | Token permission insufficient for operation |
| Scope violation | `PERMISSION_DENIED` | Agent's project not in caller's token scope |
| Session violation | `PERMISSION_DENIED` | `session`-scoped token targeting agent from different session |
