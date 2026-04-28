# ARP (Agent Registry Protocol) Test Suite

Conformance tests for the [Agent Registry Protocol (ARP)](../../arp-spec/overview.md) v0.3.

## Overview

ARP is an MCP server that manages the full lifecycle of AI agents within workspaces and provides a unified registry for A2A agent discovery and communication. It fills the gap between MCP (agent-to-tool) and A2A (agent-to-agent): neither protocol defines how to create, start, stop, or destroy agent instances. ARP does.

This suite validates ARP server implementations across 79 test cases covering:

- **MCP tool interface** — project, workspace, and agent lifecycle management
- **A2A proxy/registry** — HTTP endpoints for agent discovery and communication
- **Proxy routing** — agent resolution by ID, name, workspace, and skill
- **Auth tokens** — scoped permissions with monotonically decreasing privilege chains
- **Session isolation** — boundary enforcement between independent agent sessions
- **Direct vs. proxy** — the two-interface contract for accessing managed agents

Transport: **MCP tools** for lifecycle/messaging, **HTTP+JSON** for A2A proxy endpoints.

## Test Categories

| Category | Tests | Severity Breakdown |
|---|---|---|
| **mcp-project** | 8 | 7 required, 1 recommended |
| **mcp-workspace** | 10 | 8 required, 1 recommended, 1 optional |
| **mcp-agent-lifecycle** | 17 | 16 required, 1 recommended |
| **mcp-agent-messaging** | 8 | 6 required, 2 recommended |
| **a2a-proxy** | 9 | 7 required, 1 recommended, 1 optional |
| **a2a-proxy-routing** | 6 | 4 required, 2 recommended |
| **auth-tokens** | 9 | 7 required, 1 recommended, 1 optional |
| **auth-session-isolation** | 5 | 5 required |
| **direct-vs-proxy** | 7 | 7 required |

### MCP Project Management

Tests `project/list`, `project/register`, and `project/unregister` MCP tools. Validates project creation, listing, removal, duplicate-name rejection, and interaction with active workspaces.

### MCP Workspace Lifecycle

Tests `workspace/create`, `workspace/list`, `workspace/get`, and `workspace/destroy` MCP tools. Validates workspace creation with active status, project/status filtering, agent detail resolution, auto-spawn via `auto_agents`, and cleanup on destroy.

### MCP Agent Lifecycle

Tests `agent/spawn`, `agent/list`, `agent/status`, `agent/stop`, and `agent/restart` MCP tools. Validates AgentInstance creation with direct/proxy URLs, name overrides, prompt-based initialization, scope/permission narrowing, filtering by workspace/status/template, full AgentCard resolution, graceful shutdown, restart preservation, and multi-agent coexistence.

### MCP Agent Messaging

Tests `agent/message`, `agent/task`, and `agent/task_status` MCP tools. Validates A2A SendMessage proxying with TextPart, context_id continuation, blocking/non-blocking modes, Task creation and tracking, and history_length limiting.

### A2A Proxy Endpoints

Tests the HTTP+JSON proxy endpoints: `GET /a2a/agents`, `GET /a2a/agents/{id}/.well-known/agent-card.json`, `POST /a2a/agents/{id}/message:send`, `POST /a2a/agents/{id}/message:stream`, `GET /a2a/agents/{id}/tasks/{tid}`, and `POST /a2a/agents/{id}/tasks/{tid}:cancel`. Validates AgentCard enrichment with `metadata.arp` fields and `supportedInterfaces[0].url` pointing to the proxy.

### A2A Proxy Routing

Tests agent resolution order: by agent_id (exact match), by AgentCard.name, by workspace/instance_name composite, and by AgentSkill.tags via `POST /a2a/route/message:send`. Validates preference for ready agents over busy.

### Auth Tokens

Tests `token/create` permission requirements, session/project scope enforcement, scope narrowing on spawn (child ⊆ parent), permission lowering (no escalation), and localhost admin bypass.

### Auth Session Isolation

Tests session boundary enforcement: separate tokens cannot see each other's agents, agent/list returns only own-session agents, agent/stop on foreign-session agents returns 403, session propagation through spawn chains, and proxied A2A session enforcement.

### Direct vs. Proxy

Tests the two-interface contract: direct URL serves standard A2A v1.0 without ARP auth, proxy URL enforces token scope, proxy AgentCard includes `metadata.arp.direct_url`, and direct access intentionally bypasses ARP scoping.

## Running

```bash
# Validate the spec
spec-torture validate specs/arp/spec.yaml

# Run against an ARP server
spec-torture run specs/arp/spec.yaml --target http://localhost:9099

# Run only required tests
spec-torture run specs/arp/spec.yaml --target http://localhost:9099 --severity required

# Run only MCP project tests
spec-torture run specs/arp/spec.yaml --target http://localhost:9099 --category mcp-project

# Run only auth tests
spec-torture run specs/arp/spec.yaml --target http://localhost:9099 --tags auth

# Run only proxy tests
spec-torture run specs/arp/spec.yaml --target http://localhost:9099 --tags a2a-proxy
```

### Prerequisites

The ARP server under test must:

1. Be running and accessible at the target URL
2. Have at least one project registered with an agent template
3. Have at least one running agent (for proxy and direct-access tests)
4. Support token creation (for auth tests) — or configure `localhost_admin: true`

For auth-specific tests, the test harness must be able to create tokens with different permission levels and scopes. Configure the ARP server with `auth.localhost_admin: true` for local testing.

## Severity Levels

| Level | Count | Meaning |
|---|---|---|
| `required` | 67 | MUST implement — failure means non-compliant |
| `recommended` | 9 | SHOULD implement — failure is a warning |
| `optional` | 3 | MAY implement — informational only |

## ARP Specification Reference

- [ARP Overview](../../arp-spec/overview.md) — architecture, data model, two-interface model
- [Project Tools](../../arp-spec/tools-project.md) — project/list, project/register, project/unregister
- [Workspace Tools](../../arp-spec/tools-workspace.md) — workspace/create, workspace/list, workspace/get, workspace/destroy
- [Agent Tools](../../arp-spec/tools-agent.md) — agent/spawn, agent/list, agent/status, agent/message, agent/task, agent/task_status, agent/stop, agent/restart
- [Discovery Tools](../../arp-spec/tools-discovery.md) — agent/discover, MCP resources, A2A proxy endpoints
- [Identity & Scopes](../../arp-spec/identity-and-scopes.md) — tokens, scopes, permissions, session isolation

## Protocol Dependencies

ARP builds on two existing protocols:

- **[A2A v1.0](https://google.github.io/A2A)** — Agent-to-Agent protocol for inter-agent communication (wire protocol)
- **[MCP](https://modelcontextprotocol.io)** — Model Context Protocol for agent lifecycle management (control plane)
