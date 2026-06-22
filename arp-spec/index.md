---
title: "Agent Registry Protocol (ARP)"
created: 2026-04-06
updated: 2026-05-01
---

# Agent Registry Protocol (ARP)

A gRPC service that manages the lifecycle of A2A agents within workspaces and lets you discover them. Fills the gap between the control plane (agent lifecycle management) and A2A (agent-to-agent communication): no existing protocol defines how to create, start, stop, or destroy agent instances. ARP does.

**ARP creates, manages, and helps you discover A2A agents — then gets out of the way. A2A is how you talk to them.** ARP returns each agent's `direct_url`; clients send messages and run tasks by speaking A2A directly to that URL. ARP is never in the message path. (A single authenticated A2A ingress is available as an optional [gateway profile](profile-a2a-gateway.md).)

## Spec Documents

### Overview

Problem statement, design principles, architecture, A2A v1.0 reference tables, the "ARP locates, A2A reaches" model, protobuf data model, agent status state machine, configuration, and relationship to existing systems.

→ [Overview](overview.md)

### Project Management Service

gRPC `ProjectService` for registering and managing code repository projects — the templates that define what agents are available and how they're configured.

RPCs: `ListProjects`, `RegisterProject`, `UnregisterProject`

→ [Project Service](services-project.md)

### Workspace Management Service

gRPC `WorkspaceService` for creating and destroying isolated workspaces (git worktrees) where agents operate.

RPCs: `CreateWorkspace`, `ListWorkspaces`, `GetWorkspace`, `DestroyWorkspace`

→ [Workspace Service](services-workspace.md)

### Agent Lifecycle Service

gRPC `AgentService` for spawning, monitoring, stopping, and restarting A2A agents within workspaces. Lifecycle only — messaging is done directly via A2A at each agent's `direct_url`.

RPCs: `SpawnAgent`, `ListAgents`, `GetAgentStatus`, `StopAgent`, `RestartAgent`

→ [Agent Lifecycle Service](services-agent.md)

### Discovery & Routing Service

gRPC `DiscoveryService` for discovering agents across workspaces and the network (returns A2A `AgentCard`s). Server-streaming RPCs for real-time lifecycle monitoring. Multi-agent workflow patterns.

RPCs: `DiscoverAgents`, `WatchAgent` (server-streaming), `WatchWorkspace` (server-streaming)

→ [Discovery & Routing](services-discovery.md)

### Identity Federation & Scopes

Token-based auth model for the ARP control plane. Tokens are passed via gRPC metadata (`authorization` key) and carry project scopes and permission levels (session, project, admin). Agents inherit scoped tokens from their spawner — scope can only narrow, never widen. Session-scoped agents see only their own spawn tree; project-scoped agents manage all agents in their projects. Covers token issuance, child token flow, per-RPC enforcement, and scope narrowing on delegation. (A2A data-plane auth is the agent's own concern; OIDC federation is an optional extension.)

→ [Identity & Scopes](identity-and-scopes.md)

### A2A Gateway Profile (optional)

A non-normative, optional deployment component: a transparent A2A passthrough that fronts agents with a single authenticated ingress, enforcing ARP token scope at the edge and forwarding to each agent's `direct_url`. Not part of the core gRPC services and not required for conformance. This is where the per-agent A2A proxy endpoints, skill-based routing, and A2A `security_schemes` federation live.

→ [A2A Gateway Profile](profile-a2a-gateway.md)
