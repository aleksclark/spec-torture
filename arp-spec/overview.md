---
title: "ARP — Overview"
version: 0.5.0
created: 2026-04-06
updated: 2026-05-01
status: draft
tags: [arp, a2a, grpc, protobuf, agent-lifecycle, multi-agent, awesometree]
---

# Agent Registry Protocol (ARP) — Overview

A gRPC service that manages the full lifecycle of AI agents within workspaces and provides a unified registry for discovering A2A agents. Fills the gap between the control plane (how you **create, start, stop, and destroy** agent instances) and A2A (how agents **talk to each other**). ARP is the control plane; A2A is the wire protocol.

**One-line value proposition: ARP creates, manages, and helps you discover A2A agents — then gets out of the way. A2A is how you talk to them.**

The boundary is deliberate and sharp:

| Concern | Owned by | How |
|---------|----------|-----|
| Create / stop / restart agents | **ARP** | gRPC `AgentService` |
| Workspaces & projects | **ARP** | gRPC `WorkspaceService`, `ProjectService` |
| Discover agents, watch lifecycle | **ARP** | gRPC `DiscoveryService` (returns A2A `AgentCard`s) |
| Scoped auth for the management plane | **ARP** | gRPC `TokenService` + per-RPC enforcement |
| Send messages, run tasks, stream results | **A2A** | the agent's own A2A endpoint (`direct_url`) |

ARP never relays agent messages. Once an agent is spawned, ARP hands back its `direct_url`; clients send A2A `SendMessage` / `GetTask` to that URL with any A2A client. This keeps ARP small and its boundary with A2A unambiguous. (An optional [A2A gateway profile](profile-a2a-gateway.md) exists for deployments that need a single authenticated ingress — but it is not part of the core gRPC services and not required for conformance.)

## Problem Statement

The current agent protocol landscape has a lifecycle gap:

- **A2A** connects agents to each other. Assumes agents already exist — no spawn, no lifecycle.
- **No standard control plane** exists for agent lifecycle management.

Every team building multi-agent systems reinvents agent lifecycle management. ARP standardizes it as a gRPC service with full protobuf definitions, making agent management available to any gRPC client — CLI tools, orchestrators, dashboards, or other agents. It also exposes managed agents as a standard A2A registry, so any A2A client can discover them and then communicate with them directly.

## Design Principles

1. **Workspaces are the unit of isolation.** A workspace is a directory (typically a git worktree) where one or more agents operate. Agents within a workspace share the filesystem but have independent sessions.

2. **A2A is the wire protocol.** Every managed agent speaks A2A v1.0. The ARP server creates agents, and those agents are standard A2A agents — discoverable via `AgentCard`, communicable via `SendMessage` / `SendStreamingMessage` **directly at the agent**. ARP does not sit in the message path.

3. **gRPC is the control plane.** Agent lifecycle operations (spawn, stop, restart) are gRPC RPCs defined in protobuf. Any gRPC client can manage agents without custom integrations. HTTP/JSON access is available via gRPC-Web transcoding with `google.api.http` annotations on every RPC.

4. **Multi-agent workspaces.** A workspace can host multiple agents (e.g., a coding agent + a review agent + a test agent). Each gets its own port and `AgentCard`.

5. **ARP locates, A2A reaches.** ARP tells you *where* an agent is (its `direct_url`, returned by `SpawnAgent` and advertised in the agent's `AgentCard`); A2A is *how* you reach it. Clients connect to the agent's A2A endpoint directly with any A2A client. ARP itself only speaks A2A for discovery (fetching `AgentCard`s) and health checks — never to relay messages or tasks. Deployments that need a single authenticated ingress can put the optional [A2A gateway](profile-a2a-gateway.md) in front of agents, but that is a deployment choice, not part of the core protocol.

6. **Backend-agnostic.** The spec defines the interface, not the implementation. Backends can be local processes, Docker containers, remote VMs, or cloud services.

7. **Scoped authority flows downward.** Every caller authenticates with a token passed via gRPC metadata (`authorization` key). Tokens carry project scopes and a permission level. Agents inherit tokens from their spawner — scope can only narrow, permission can only lower. This creates a monotonically decreasing privilege chain with no escalation path. See [Identity & Scopes](identity-and-scopes.md).

## gRPC Service Groups

| Group | Service | RPCs | Spec |
|-------|---------|------|------|
| Project Management | `ProjectService` | `ListProjects`, `RegisterProject`, `UnregisterProject` | [services-project.md](services-project.md) |
| Workspace Management | `WorkspaceService` | `CreateWorkspace`, `ListWorkspaces`, `GetWorkspace`, `DestroyWorkspace` | [services-workspace.md](services-workspace.md) |
| Agent Lifecycle | `AgentService` | `SpawnAgent`, `ListAgents`, `GetAgentStatus`, `StopAgent`, `RestartAgent` | [services-agent.md](services-agent.md) |
| Discovery & Routing | `DiscoveryService` | `DiscoverAgents`, `WatchAgent`, `WatchWorkspace` | [services-discovery.md](services-discovery.md) |
| Identity & Scopes | `TokenService` | `CreateToken`; scope enforcement | [identity-and-scopes.md](identity-and-scopes.md) |

> Messaging and task RPCs are intentionally **absent** from `AgentService`. Sending a message, creating a task, or polling task status is done by speaking A2A directly to the agent's `direct_url` — see [services-agent.md](services-agent.md#talking-to-an-agent-a2a-not-arp). This is the single most important boundary in the spec: **ARP = lifecycle + registry; A2A = messaging.**

## A2A v1.0 Reference

This spec builds on the A2A v1.0 protocol (`lf.a2a.v1`). Key types and RPCs referenced throughout:

### Types Used

| A2A Type | Description | Key Fields |
|----------|-------------|------------|
| `AgentCard` | Agent's discovery document | `name`, `description`, `version`, `supported_interfaces[]`, `capabilities`, `skills[]`, `default_input_modes[]`, `default_output_modes[]`, `security_schemes`, `metadata` |
| `AgentInterface` | Service endpoint binding | `url`, `transport` (JSONRPC, GRPC, HTTP_JSON) |
| `AgentCapabilities` | What the agent supports | `streaming`, `push_notifications`, `state_transition_history`, `extensions[]` |
| `AgentSkill` | A capability the agent offers | `id`, `name`, `description`, `tags[]`, `examples[]`, `input_modes[]`, `output_modes[]` |
| `Task` | A tracked unit of work | `id`, `context_id`, `status`, `history[]`, `artifacts[]`, `metadata` |
| `TaskStatus` | Current state + timestamp | `state` (TaskState enum), `timestamp`, `message` |
| `TaskState` | Lifecycle enum | `SUBMITTED`, `WORKING`, `INPUT_REQUIRED`, `COMPLETED`, `CANCELED`, `FAILED`, `REJECTED`, `AUTH_REQUIRED` |
| `Message` | A communication unit | `message_id`, `role` (USER, AGENT), `parts[]`, `task_id`, `context_id`, `reference_task_ids[]`, `metadata` |
| `Part` | Content within a message | oneof: `TextPart` {`text`}, `FilePart` {`file`}, `DataPart` {`data`} |
| `Artifact` | Output produced by a task | `artifact_id`, `name`, `description`, `parts[]`, `metadata` |
| `SendMessageRequest` | Request to send a message | `message` (Message), `configuration` (SendMessageConfiguration) |
| `SendMessageResponse` | Response from agent | oneof: `task` (Task) or `message` (Message) |
| `StreamResponse` | Streaming response event | oneof: `task` (Task) or `message` (Message) |

### RPCs Used

These are **A2A** RPCs that clients invoke **directly on the agent** (at its `direct_url`), not ARP RPCs. ARP returns the `direct_url`; the client takes it from there. ARP itself only calls the discovery endpoint (`GET /.well-known/agent-card.json`).

| A2A RPC | HTTP Binding | Description |
|---------|-------------|-------------|
| `SendMessage` | `POST /message:send` | Send a message, get sync response |
| `SendStreamingMessage` | `POST /message:stream` | Send a message, get SSE stream |
| `GetTask` | `GET /tasks/{id}` | Get current task state |
| `ListTasks` | `GET /tasks` | List tasks with filters |
| `CancelTask` | `POST /tasks/{id}:cancel` | Cancel a running task |
| `SubscribeToTask` | `GET /tasks/{id}:subscribe` | SSE stream of task updates |
| `GetExtendedAgentCard` | `GET /extendedAgentCard` | Authenticated extended card |

### Discovery

| Endpoint | Description |
|----------|-------------|
| `GET /.well-known/agent-card.json` | Public `AgentCard` discovery (RFC 8615) |
| `GET /extendedAgentCard` | Authenticated `AgentCard` with additional capabilities/skills |

## Architecture

```
┌────────────────────────────────────────────────────────────┐
│                     gRPC CLIENTS                            │
│  (CLI tools, orchestrators, dashboards, other agents)       │
│                                                             │
│  Lifecycle:  SpawnAgent, StopAgent, RestartAgent, ...       │
│  Registry:   DiscoverAgents, WatchAgent, WatchWorkspace     │
└──────────┬─────────────────────────────────────────────────┘
           │ gRPC (H2) or gRPC-Web (HTTP/1.1 transcoding)
           ▼
┌────────────────────────────────────────────────────────────┐
│                    ARP SERVER                                │
│              (control plane + A2A registry)                  │
│                                                             │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │  Workspace   │  │    Agent     │  │   A2A Registry  │   │
│  │  Manager     │  │  Supervisor  │  │  (AgentCards)   │   │
│  └──────┬──────┘  └──────┬───────┘  └────────┬────────┘   │
│         │                │                     │            │
│         │      ┌─────────▼─────────┐           │ fetch      │
│         │      │  Process Backend  │           │ AgentCard  │
│         │      │  (local/docker/   │           │ (discovery │
│         │      │   remote/cloud)   │           │  only)     │
│         │      └─────────┬─────────┘           │            │
└─────────┼────────────────┼─────────────────────┼────────────┘
          │                │                     │
          ▼                ▼                     │
    ┌──────────┐    ┌──────────────┐             │
    │Workspace │    │ Agent        │ ◀───────────┘
    │Directory │    │ Processes    │
    │(worktree)│    │              │     A2A messaging goes
    │          │    │ :9100 crush  │ ◀── DIRECT, client→agent
    │          │    │ :9101 claude │     (never through ARP)
    │          │    │ :9102 review │
    └──────────┘    └──────┬───────┘
                           ▲
            ┌──────────────┘  client uses direct_url from
            │  the SpawnAgent response / DiscoverAgents card
   ┌────────┴────────┐
   │  A2A CLIENTS    │  SendMessage / GetTask / stream
   │ (any A2A impl)  │
   └─────────────────┘
```

> An optional [A2A gateway](profile-a2a-gateway.md) may be deployed in front of
> the agents to provide a single authenticated A2A ingress. It is a separate
> component, not part of the ARP gRPC services, and is omitted here.

## Reaching Agents: ARP Locates, A2A Reaches

ARP's job ends at *locating* an agent. Every managed agent is a standard A2A v1.0 server bound to its own port; clients reach it by speaking A2A directly to its `direct_url`.

### Direct access (the only path in core ARP)

`SpawnAgent` / `GetAgentStatus` / `ListAgents` / `DiscoverAgents` all return the agent's `direct_url` (and an enriched `AgentCard` whose first `supported_interfaces` entry is that URL). The client then uses any A2A client:

```
direct_url: http://localhost:{agent_port}

A2A v1.0 HTTP+JSON endpoints (served by the agent itself, not ARP):
  GET  /.well-known/agent-card.json       AgentCard discovery (RFC 8615)
  POST /message:send                       SendMessage → SendMessageResponse
  POST /message:stream                     SendStreamingMessage → stream StreamResponse
  GET  /tasks/{id}                         GetTask → Task
  GET  /tasks                              ListTasks → ListTasksResponse
  POST /tasks/{id}:cancel                  CancelTask → Task
  GET  /tasks/{id}:subscribe               SubscribeToTask → stream StreamResponse
  GET  /extendedAgentCard                  GetExtendedAgentCard → AgentCard
```

Typical flow:

```
1. ARP:  SpawnAgent(workspace, template)  → AgentInstance{ direct_url }   (gRPC)
2. A2A:  POST {direct_url}/message:send   → Task                          (direct)
3. A2A:  GET  {direct_url}/tasks/{id}     → Task (poll to terminal)       (direct)
4. ARP:  StopAgent(agent_id)                                              (gRPC)
```

Because ARP isn't in the message path, there is no proxy hop, no duplicated A2A surface, and no ambiguity about which interface "really" talks to the agent. If an agent restarts on a new port, re-read its `direct_url` from ARP (or subscribe via `WatchAgent`).

### Optional: a single ingress via the A2A gateway

Some deployments want one authenticated entry point in front of agents — e.g. agents on a private network, centralized auth/audit, or routing by skill without the client knowing agent IDs. That is provided by the optional **[A2A gateway profile](profile-a2a-gateway.md)**, a transparent A2A passthrough that enforces ARP token scope at the edge and forwards requests to each agent's `direct_url`. It is:

- **Not** part of the core ARP gRPC services.
- **Not** required for conformance.
- A deployment component you add when you need it.

When a gateway is present, ARP populates `AgentInstance.proxy_url` and adds the gateway URL as a second `supported_interfaces` entry; otherwise `proxy_url` is empty.

### Agent Card Enrichment

When ARP returns an agent's `AgentCard` (via `GetAgentStatus`, `ListAgents`, or `DiscoverAgents`), it enriches the card's `metadata` with an `arp` block carrying lifecycle context. The first `supported_interfaces` entry is the agent's **direct** URL:

```json
{
  "name": "crush-coder",
  "description": "Crush AI coding assistant",
  "version": "1.0.0",
  "supported_interfaces": [
    { "url": "http://localhost:9100", "transport": "HTTP_JSON" }
  ],
  "capabilities": { "streaming": true },
  "skills": [
    {
      "id": "code",
      "name": "Code",
      "description": "Write, review, and debug code",
      "tags": ["coding", "debugging", "refactoring"],
      "examples": ["Implement OAuth2 login flow", "Fix the null pointer in auth.rs"]
    }
  ],
  "default_input_modes": ["text/plain"],
  "default_output_modes": ["text/plain"],
  "metadata": {
    "arp": {
      "agent_id": "coder-abc123",
      "workspace": "feat-auth",
      "project": "myapp",
      "template": "crush",
      "status": "ready",
      "direct_url": "http://localhost:9100",
      "started_at": "2026-04-06T10:30:00Z"
    }
  }
}
```

`supported_interfaces[0].url` and `metadata.arp.direct_url` both point at the agent's own endpoint — that is where clients send A2A traffic. When the optional gateway is deployed, a second interface (the gateway URL) is appended and `metadata.arp.proxy_url` is set; the direct interface remains first.

## Protobuf Data Model

All ARP data types are defined as proto3 messages in `arp/v1/arp.proto`. The following are the core message types; full protobuf definitions are in the individual service specs.

### Project

A project is a code repository with configuration for how agents should operate within it.

```protobuf
message Project {
  string name = 1 [(google.api.field_behavior) = REQUIRED];     // Unique project identifier
  string repo = 2 [(google.api.field_behavior) = REQUIRED];     // Path to the git repository
  string branch = 3;                                             // Default branch (default: "main")
  repeated AgentTemplate agents = 4;                             // Agent templates available
  ProjectContext context = 5;                                    // Context configuration
}

message ProjectContext {
  repeated string files = 1;           // Context files fed to agents
  repeated string repo_includes = 2;   // Glob patterns for repo context
  int64 max_bytes = 3;                 // Max context size
}
```

### Workspace

A workspace is an isolated working directory (typically a git worktree) that hosts one or more agents.

```protobuf
message Workspace {
  string name = 1 [(google.api.field_behavior) = REQUIRED];     // Unique workspace identifier
  string project = 2 [(google.api.field_behavior) = REQUIRED];  // Parent project name
  string dir = 3;                                                // Absolute path to workspace directory
  WorkspaceStatus status = 4;                                    // Workspace lifecycle state
  repeated AgentInstance agents = 5;                             // Agents running in this workspace
  google.protobuf.Timestamp created_at = 6;                     // Creation timestamp
  google.protobuf.Struct metadata = 7;                          // Extensible metadata
}

enum WorkspaceStatus {
  WORKSPACE_STATUS_UNSPECIFIED = 0;
  WORKSPACE_STATUS_ACTIVE = 1;
  WORKSPACE_STATUS_INACTIVE = 2;
}
```

### AgentTemplate

Defines how to spawn a particular type of agent. Templates are configured per-project or globally. The `a2a_card_config` field populates the `AgentCard` that ARP generates or enriches for this agent.

```protobuf
message AgentTemplate {
  string name = 1 [(google.api.field_behavior) = REQUIRED];     // Template name (e.g., "crush", "claude-code")
  string command = 2 [(google.api.field_behavior) = REQUIRED];  // Command to start the agent process
  string port_env = 3;                                           // Env var name for port assignment
  HealthCheckConfig health_check = 4;                            // Health check configuration
  map<string, string> env = 5;                                   // Additional environment variables
  repeated string capabilities = 6;                              // Declared capabilities
  A2ACardConfig a2a_card_config = 7;                             // Fields for AgentCard generation
}

message HealthCheckConfig {
  string path = 1;            // Health endpoint (e.g., "/.well-known/agent-card.json")
  int32 interval_ms = 2;     // Check interval (default: 5000)
  int32 timeout_ms = 3;      // Timeout per check (default: 3000)
  int32 retries = 4;         // Retries before marking unhealthy (default: 3)
}

message A2ACardConfig {
  string name = 1;                                    // AgentCard.name override
  string description = 2;                             // AgentCard.description
  repeated lf.a2a.v1.AgentSkill skills = 3;           // AgentCard.skills[]
  repeated string input_modes = 4;                    // AgentCard.default_input_modes[]
  repeated string output_modes = 5;                   // AgentCard.default_output_modes[]
  lf.a2a.v1.AgentCapabilities capabilities = 6;       // AgentCard.capabilities
}
```

### AgentInstance

A running agent within a workspace.

```protobuf
message AgentInstance {
  string id = 1;                                      // Unique instance identifier (server-generated)
  string template = 2;                                // Template name used to spawn
  string workspace = 3;                               // Parent workspace name
  AgentStatus status = 4;                             // Current lifecycle state
  int32 port = 5;                                     // Assigned port number
  string direct_url = 6;                              // Agent's A2A endpoint — reach the agent here
  string proxy_url = 7;                               // Optional gateway URL; empty unless a gateway is deployed
  int32 pid = 8;                                      // Process ID (if local backend)
  // field 9 reserved (was context_id — an A2A conversation concept ARP no longer tracks)
  lf.a2a.v1.AgentCard a2a_agent_card = 10;            // Resolved A2A AgentCard (with ARP metadata)
  string token_id = 11;                               // ARP token issued to this agent
  string session_id = 12;                             // Session this agent belongs to
  string spawned_by = 13;                             // Token ID of the caller that spawned this agent
  google.protobuf.Timestamp started_at = 14;          // Start timestamp
  google.protobuf.Struct metadata = 15;               // Extensible metadata
}

// ARP lifecycle states — track the agent PROCESS, distinct from A2A TaskState
// which tracks task execution inside the agent.
enum AgentStatus {
  AGENT_STATUS_UNSPECIFIED = 0;
  AGENT_STATUS_STARTING = 1;     // Process launched, waiting for health check
  AGENT_STATUS_READY = 2;        // Health check passed, reachable for A2A
  AGENT_STATUS_BUSY = 3;         // Optional: task-level activity observed (gateway deployments)
  AGENT_STATUS_ERROR = 4;        // Health check failed or process crashed
  AGENT_STATUS_STOPPING = 5;     // Graceful shutdown initiated (SIGTERM sent)
  AGENT_STATUS_STOPPED = 6;      // Process terminated
}
```

### AgentStatus State Machine

```
             spawn
               │
               ▼
          ┌─────────┐
          │starting  │──── health check fails (retries exhausted) ──→ error
          └────┬─────┘                                                  │
               │ health check passes                              restart│
               ▼                                                         │
          ┌─────────┐                                                   │
     ┌──→ │  ready   │ ──── crash ──→ error ────────── restart ─────────┘
     │    └────┬─────┘
     │         │ terminate
     │         ▼
     │    ┌─────────┐         ┌─────────┐
     └──→ │stopping  │ ──────→│ stopped  │
          └─────────┘  grace  └─────────┘
                       period

(optional, gateway only)   ready ⇄ busy   while task-level activity is observed
```

Note: ARP `AgentStatus` tracks the **process** lifecycle. A2A `TaskState` tracks **task** execution within the agent — and because core ARP is not in the message path, it does not observe task state. The canonical core lifecycle is `starting → ready → stopping → stopped` (with `error`). `AGENT_STATUS_BUSY` is reserved for deployments where ARP *does* observe task activity (e.g. via the optional [A2A gateway](profile-a2a-gateway.md)); core ARP leaves reachable agents in `AGENT_STATUS_READY`. An agent in `READY` may still have completed or running tasks in its A2A history — query the agent directly to see them.

## Configuration

The ARP server is configured via a JSON file:

```json
{
  "port": 9099,
  "port_range": { "min": 9100, "max": 9199 },
  "state_dir": "~/.config/arp/",
  "templates": {
    "crush": {
      "command": "crush serve",
      "port_env": "A2A_PORT",
      "health_check": { "path": "/.well-known/agent-card.json", "interval_ms": 5000 },
      "a2a_card_config": {
        "name": "Crush",
        "description": "AI coding assistant",
        "skills": [
          {
            "id": "code",
            "name": "Code",
            "description": "Write, review, and debug code",
            "tags": ["coding"]
          }
        ],
        "capabilities": {
          "streaming": true,
          "push_notifications": false,
          "state_transition_history": false
        }
      }
    },
    "claude-code": {
      "command": "claude --a2a",
      "port_env": "A2A_PORT",
      "health_check": { "path": "/.well-known/agent-card.json", "interval_ms": 5000 }
    }
  },
  "process": {
    "grace_period_ms": 5000,
    "restart_delay_ms": 2000,
    "auto_restart": true,
    "max_restart_attempts": 3
  },
  "a2a": {
    "registry_enabled": true,
    "proxy_enabled": true
  },
  "auth": {
    "mode": "local",
    "localhost_admin": true,
    "federation": []
  }
}
```

## Relationship to Existing Systems

### Awesometree

ARP generalizes awesometree's workspace and agent supervisor model. An awesometree-backed implementation would:
- Use awesometree's `Manager` for workspace creation (git worktrees, WM tags)
- Extend the agent `Supervisor` to manage multiple agents per workspace (currently one)
- Expose both the existing REST API and the new gRPC service
- Serve the A2A `AgentCard` registry (discovery) from the existing HTTP server (port 9099)
- Maintain backward compatibility with the current single-agent-per-workspace model

### A2A

ARP is an A2A **registry and discovery** layer plus a lifecycle controller — and a minimal A2A *client* only for reading `AgentCard`s and health-checking agents. It is **not** an A2A server for messaging: it never relays `SendMessage`/`GetTask`. A2A defines how to *talk* to agents; ARP defines how to *create, manage, and find* them. The two are complementary and cleanly separated — ARP creates agents that speak standard A2A v1.0, then points clients at them. (Centralized A2A ingress, when wanted, is the optional [gateway profile](profile-a2a-gateway.md), not the core protocol.)

### gRPC + HTTP Transcoding

ARP's control plane is a set of gRPC services defined in protobuf. Every RPC carries `google.api.http` annotations, so the same server can serve both native gRPC clients (high-performance, streaming) and HTTP/JSON clients (via gRPC-Web transcoding or an Envoy/grpc-gateway sidecar). This mirrors the pattern used by A2A v1.0 itself, which defines its RPCs in protobuf with HTTP bindings.
