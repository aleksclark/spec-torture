---
title: "ARP — Discovery, Routing & Patterns"
version: 0.5.0
created: 2026-04-06
updated: 2026-05-01
status: draft
tags: [arp, grpc, protobuf, a2a, discovery, agentcard, patterns]
---

# Discovery, Routing & Patterns

gRPC service for discovering agents, server-streaming RPCs for real-time monitoring, and multi-agent patterns showing how the service groups compose.

## Service Definition

```protobuf
syntax = "proto3";

package arp.v1;

import "google/api/annotations.proto";
import "google/api/field_behavior.proto";
import "lf/a2a/v1/a2a.proto";

// DiscoveryService provides agent discovery, routing, and real-time monitoring.
service DiscoveryService {
  // DiscoverAgents discovers available agents across workspaces or on the network.
  // Returns AgentCards from managed workspaces (local) or by probing
  // /.well-known/agent-card.json at given URLs (network).
  rpc DiscoverAgents(DiscoverAgentsRequest) returns (DiscoverAgentsResponse) {
    option (google.api.http) = {
      get: "/v1/discover"
    };
  }

  // WatchAgent streams real-time status updates for an agent instance.
  // The server sends an AgentEvent whenever the agent's status, port,
  // or AgentCard changes. Replaces the old MCP resource subscription model.
  rpc WatchAgent(WatchAgentRequest) returns (stream AgentEvent) {
    option (google.api.http) = {
      get: "/v1/agents/{agent_id}:watch"
    };
  }

  // WatchWorkspace streams real-time updates for a workspace.
  // The server sends a WorkspaceEvent whenever an agent is spawned,
  // stopped, or changes status within the workspace.
  rpc WatchWorkspace(WatchWorkspaceRequest) returns (stream WorkspaceEvent) {
    option (google.api.http) = {
      get: "/v1/workspaces/{workspace_name}:watch"
    };
  }
}
```

## Messages

### DiscoverAgents

```protobuf
enum DiscoveryScope {
  DISCOVERY_SCOPE_UNSPECIFIED = 0;
  DISCOVERY_SCOPE_LOCAL = 1;    // Managed agents only
  DISCOVERY_SCOPE_NETWORK = 2;  // Probe URLs for AgentCards
}

message DiscoverAgentsRequest {
  // Search scope: LOCAL for managed agents only,
  // NETWORK probes URLs for AgentCards (default: LOCAL).
  DiscoveryScope scope = 1;

  // Filter by AgentSkill.tags (e.g., "code", "review", "test").
  string capability = 2;

  // Base URLs to probe for /.well-known/agent-card.json (for NETWORK scope).
  repeated string urls = 3;
}

message DiscoverAgentsResponse {
  repeated lf.a2a.v1.AgentCard agent_cards = 1;
}
```

**Local scope:** Returns `AgentCard` for every managed agent with status `AGENT_STATUS_READY` or `AGENT_STATUS_BUSY`. Cards are enriched with `metadata.arp` (see [Agent Card Enrichment](overview.md#agent-card-enrichment)).

**Network scope:** For each URL in `urls`, issues `GET {url}/.well-known/agent-card.json` (A2A discovery per RFC 8615). Returns successfully fetched `AgentCard` messages. Non-managed agents will not have `metadata.arp`.

**Capability filtering:** When `capability` is set, only returns agents whose `AgentCard.skills[]` contains at least one `AgentSkill` with a matching `tags[]` entry.

### WatchAgent

Server-streaming RPC for real-time agent monitoring. Replaces the old subscription-based resource model.

```protobuf
message WatchAgentRequest {
  // Agent instance ID.
  string agent_id = 1 [(google.api.field_behavior) = REQUIRED];
}

message AgentEvent {
  // The event type.
  AgentEventType event_type = 1;

  // Current agent instance state.
  AgentInstance agent = 2;

  // The agent's resolved A2A AgentCard (included on CARD_UPDATED events).
  lf.a2a.v1.AgentCard agent_card = 3;
}

enum AgentEventType {
  AGENT_EVENT_TYPE_UNSPECIFIED = 0;
  AGENT_EVENT_TYPE_STATUS_CHANGED = 1;  // starting→ready, ready→busy, etc.
  AGENT_EVENT_TYPE_CARD_UPDATED = 2;    // AgentCard changed (restart, port change)
  AGENT_EVENT_TYPE_STOPPED = 3;         // Agent process terminated
}
```

**Stream behavior:**
- On connection, sends an initial `STATUS_CHANGED` event with current state
- Subsequent events sent on status changes (`AGENT_STATUS_STARTING` → `AGENT_STATUS_READY`, `AGENT_STATUS_READY` → `AGENT_STATUS_BUSY`, etc.)
- `CARD_UPDATED` sent when the agent restarts (port may change) or the AgentCard is updated
- `STOPPED` sent when the agent process terminates; stream closes after this event

### WatchWorkspace

Server-streaming RPC for real-time workspace monitoring.

```protobuf
message WatchWorkspaceRequest {
  // Workspace name.
  string workspace_name = 1 [(google.api.field_behavior) = REQUIRED];
}

message WorkspaceEvent {
  // The event type.
  WorkspaceEventType event_type = 1;

  // Full workspace state.
  Workspace workspace = 2;

  // The agent involved in this event (for AGENT_* events).
  AgentInstance agent = 3;
}

enum WorkspaceEventType {
  WORKSPACE_EVENT_TYPE_UNSPECIFIED = 0;
  WORKSPACE_EVENT_TYPE_AGENT_SPAWNED = 1;         // New agent spawned
  WORKSPACE_EVENT_TYPE_AGENT_STATUS_CHANGED = 2;  // Agent status changed
  WORKSPACE_EVENT_TYPE_AGENT_STOPPED = 3;          // Agent stopped
  WORKSPACE_EVENT_TYPE_WORKSPACE_DESTROYED = 4;    // Workspace destroyed; stream closes
}
```

**Stream behavior:**
- On connection, sends an initial `AGENT_SPAWNED` event for each existing agent
- Subsequent events sent as agents are spawned, change status, or stop
- `WORKSPACE_DESTROYED` sent when the workspace is destroyed; stream closes after this event

## HTTP Registry Endpoint

The registry is also reachable over plain HTTP as the transcoding of `DiscoverAgents` — a **read-only discovery** surface, not a proxy:

```
GET /v1/discover                  Filtered discovery (gRPC transcoding of DiscoverAgents)
GET /a2a/agents                   List enriched AgentCards for ready agents (alias)
```

These return A2A `AgentCard` JSON (with `metadata.arp.direct_url`). A client uses them to *find* an agent, then sends A2A messages **directly to that agent's `direct_url`**.

Skill-based message **routing** (e.g. `POST /a2a/route/message:send`) and any A2A message forwarding are deliberately **not** part of `DiscoveryService` — they require ARP to sit in the message path. That capability lives in the optional [A2A gateway profile](profile-a2a-gateway.md).

## Multi-Agent Workspace Patterns

These patterns show how the gRPC services compose. The rule throughout: **ARP for lifecycle, discovery, and watching; A2A (at each agent's `direct_url`) for messages and tasks.** `grpcurl` shows the ARP calls; `curl` shows the A2A calls.

### Pattern 1: Coder + Reviewer

Two agents in one workspace — one writes code, the other reviews it.

```bash
# Lifecycle via ARP
grpcurl -d '{"name":"feat-auth","project":"myapp"}' \
  localhost:9099 arp.v1.WorkspaceService/CreateWorkspace
CODER=$(grpcurl -d '{"workspace":"feat-auth","template":"crush","name":"coder"}' \
  localhost:9099 arp.v1.AgentService/SpawnAgent | jq -r .directUrl)
REVIEWER=$(grpcurl -d '{"workspace":"feat-auth","template":"crush","name":"reviewer"}' \
  localhost:9099 arp.v1.AgentService/SpawnAgent | jq -r .directUrl)

# Messaging via A2A — straight to each agent
curl -X POST "$CODER/message:send" -H 'Content-Type: application/json' \
  -d '{"message":{"role":"ROLE_USER","parts":[{"text_part":{"text":"Implement OAuth2 login flow"}}]}}'
# ... poll GET "$CODER/tasks/{id}" until status.state is terminal ...
curl -X POST "$REVIEWER/message:send" -H 'Content-Type: application/json' \
  -d '{"message":{"role":"ROLE_USER","parts":[{"text_part":{"text":"Review the changes in this workspace"}}]}}'
```

### Pattern 2: Parallel Implementation

Multiple agents work on different parts of a codebase simultaneously.

```bash
grpcurl -d '{"name":"refactor","project":"myapp"}' \
  localhost:9099 arp.v1.WorkspaceService/CreateWorkspace
BACKEND=$(grpcurl -d '{"workspace":"refactor","template":"crush","name":"backend"}' \
  localhost:9099 arp.v1.AgentService/SpawnAgent | jq -r .directUrl)
FRONTEND=$(grpcurl -d '{"workspace":"refactor","template":"crush","name":"frontend"}' \
  localhost:9099 arp.v1.AgentService/SpawnAgent | jq -r .directUrl)

# Assign tasks via A2A (each returns a Task); poll GET {url}/tasks/{id} to track
curl -X POST "$BACKEND/message:send"  -H 'Content-Type: application/json' \
  -d '{"message":{"role":"ROLE_USER","parts":[{"text_part":{"text":"Refactor the API layer to use GraphQL"}}]}}'
curl -X POST "$FRONTEND/message:send" -H 'Content-Type: application/json' \
  -d '{"message":{"role":"ROLE_USER","parts":[{"text_part":{"text":"Update React components for the new GraphQL API"}}]}}'
```

### Pattern 3: Supervisor + Workers

A gRPC client (or agent) acts as supervisor, spawning specialist agents dynamically and watching their lifecycle via ARP while delegating work via A2A:

```bash
grpcurl -d '{"name":"big-refactor","project":"myapp"}' \
  localhost:9099 arp.v1.WorkspaceService/CreateWorkspace
PLANNER=$(grpcurl -d '{"workspace":"big-refactor","template":"crush","name":"planner"}' \
  localhost:9099 arp.v1.AgentService/SpawnAgent | jq -r .directUrl)

# Plan via A2A
curl -X POST "$PLANNER/message:send" -H 'Content-Type: application/json' \
  -d '{"message":{"role":"ROLE_USER","parts":[{"text_part":{"text":"Break this into subtasks: ..."}}]}}'

# Spawn workers via ARP, delegate via A2A
grpcurl -d '{"workspace":"big-refactor","template":"crush","name":"worker-1"}' \
  localhost:9099 arp.v1.AgentService/SpawnAgent
grpcurl -d '{"workspace":"big-refactor","template":"crush","name":"worker-2"}' \
  localhost:9099 arp.v1.AgentService/SpawnAgent

# Monitor process lifecycle via ARP (server-streaming); task progress via A2A GetTask
grpcurl -d '{"agent_id":"worker-1-xxx"}' \
  localhost:9099 arp.v1.DiscoveryService/WatchAgent
```

### Pattern 4: External A2A Client (discover via ARP, talk via A2A)

An external client uses ARP only to find agents, then speaks A2A directly:

```bash
# Discover via ARP registry (HTTP)
curl http://arp-server:9099/a2a/agents
# → Array of AgentCard objects; take supported_interfaces[0].url (== metadata.arp.direct_url)

# Talk directly to the agent via A2A — no ARP in the path
curl -X POST http://localhost:9100/message:send -H 'Content-Type: application/json' \
  -d '{"message":{"role":"ROLE_USER","parts":[{"text_part":{"text":"Fix the auth bug"}}]}}'
# → SendMessageResponse: { "task": { "id": "task-abc123", "status": { "state": "TASK_STATE_WORKING" } } }

curl http://localhost:9100/tasks/task-abc123
# → Task: { "id": "task-abc123", "status": { "state": "TASK_STATE_COMPLETED" }, "artifacts": [...] }
```

> If the deployment fronts agents with the optional [A2A gateway](profile-a2a-gateway.md), the client uses each card's `metadata.arp.proxy_url` (a second `supported_interfaces` entry) instead of `direct_url`; the gateway enforces token scope and forwards to the agent unchanged.

## gRPC Status Codes

| Condition | gRPC Status | Description |
|-----------|-------------|-------------|
| Agent not found | `NOT_FOUND` | WatchAgent with unknown `agent_id` |
| Workspace not found | `NOT_FOUND` | WatchWorkspace with unknown workspace name |
| Network probe failed | `UNAVAILABLE` | DiscoverAgents NETWORK scope URL unreachable |
| Permission denied | `PERMISSION_DENIED` | Token permission insufficient |
| Scope violation | `PERMISSION_DENIED` | Target not in caller's token scope |
