---
title: "ARP — A2A Gateway Profile (optional)"
version: 0.5.0
created: 2026-05-01
updated: 2026-05-01
status: draft
tags: [arp, a2a, gateway, proxy, optional, profile]
---

# A2A Gateway Profile (optional)

> **Status: optional, non-normative.** Nothing in this document is part of the
> core ARP gRPC services ([`AgentService`](services-agent.md),
> [`DiscoveryService`](services-discovery.md), etc.) and **none of it is
> required for ARP conformance.** A conformant ARP server may implement zero of
> the endpoints described here. The gateway is a deployment component you add
> only when you need a single authenticated A2A ingress.

## Why a gateway exists (and why it's separate)

Core ARP follows one rule: **ARP locates, A2A reaches.** `SpawnAgent` returns a
`direct_url`; clients send A2A messages straight to the agent. ARP is never in
the message path. This keeps the protocol small and its boundary with A2A sharp.

Some deployments, however, genuinely need a single front door for the **A2A data
plane** — for reasons that are about *operations*, not *protocol*:

- Agents run on a private network and aren't directly reachable by clients.
- Centralized authentication, authorization, rate-limiting, or audit of A2A
  traffic is required.
- Clients want to address agents by **skill/capability** rather than by knowing
  a specific `direct_url`.

The gateway serves those needs **without** changing ARP. It is a transparent A2A
reverse proxy: it speaks A2A on the front, enforces ARP token scope at the edge,
and forwards each request unchanged to the target agent's `direct_url`. It does
**not** add new task semantics, wrap A2A responses, or define new message types.

Keeping it out of `AgentService` is deliberate: folding A2A messaging back into
the gRPC control plane is exactly what makes ARP and A2A entangle. The gateway is
where centralized ingress belongs, clearly labeled as optional.

## What the gateway provides

When a gateway is deployed:

- ARP populates `AgentInstance.proxy_url = {gateway}/a2a/agents/{agent_id}`.
- The enriched `AgentCard` lists the **direct** interface first and appends the
  gateway interface; `metadata.arp.proxy_url` is set alongside `direct_url`.
- Clients may use either URL. Direct is lowest-latency; the gateway adds
  centralized auth/routing.

### Per-agent passthrough endpoints

The gateway mirrors the A2A v1.0 HTTP+JSON bindings under a per-agent prefix and
forwards verbatim to the agent's `direct_url`:

```
GET  /a2a/agents/{agent_id}/.well-known/agent-card.json  → {direct_url}/.well-known/agent-card.json
POST /a2a/agents/{agent_id}/message:send                  → {direct_url}/message:send
POST /a2a/agents/{agent_id}/message:stream                → {direct_url}/message:stream   (SSE relayed)
GET  /a2a/agents/{agent_id}/tasks/{task_id}               → {direct_url}/tasks/{task_id}
GET  /a2a/agents/{agent_id}/tasks                         → {direct_url}/tasks
POST /a2a/agents/{agent_id}/tasks/{task_id}:cancel        → {direct_url}/tasks/{task_id}:cancel
GET  /a2a/agents/{agent_id}/tasks/{task_id}:subscribe     → {direct_url}/tasks/{task_id}:subscribe (SSE)
GET  /a2a/agents/{agent_id}/extendedAgentCard             → {direct_url}/extendedAgentCard
```

Request and response bodies are A2A-native and **unmodified** by the gateway.

### Skill-based routing

```
POST /a2a/route/message:send     Route an A2A SendMessage to a matching agent
```

The gateway resolves a target agent, then forwards the body unchanged. Resolution
order:

1. **By agent_id** — exact match to a managed agent instance.
2. **By name** — matches `AgentCard.name`.
3. **By workspace/name** — `{workspace}/{instance_name}` composite key.
4. **By skill** — matches `AgentSkill.tags` on the agent's `AgentCard.skills[]`.

If multiple agents match, the gateway prefers `AGENT_STATUS_READY` over
`AGENT_STATUS_BUSY`. (This is the only context in which `AGENT_STATUS_BUSY` is
meaningful — a gateway that observes A2A task activity may surface it; core ARP
does not.)

## Scope enforcement at the edge

For every gateway request the server:

1. Extracts the bearer token from the `Authorization` HTTP header.
2. Resolves the target agent (by id, name, or skill match).
3. Checks the agent's project is within the token's scope.
4. For `SESSION` permission, checks the agent's `session_id` matches the token's.
5. If authorized, forwards to the agent's `direct_url`.
6. If denied, returns **HTTP 403** with an A2A-compatible error body.

This mirrors the per-RPC scope rules in [Identity & Scopes](identity-and-scopes.md)
for the proxied data plane. Direct access to an agent's own port is **not**
gated by the gateway — that is plain A2A, secured (if at all) by the agent's own
`security_schemes`.

## Identity federation (A2A security schemes)

When fronting agents that declare A2A `AgentCard.security_schemes`
(`openIdConnect`, `oauth2`, `apiKey`, `http`), the gateway may map external A2A
auth to internal ARP tokens:

```json
{
  "gateway": {
    "listen": ":9099",
    "a2a_federation": [
      {
        "scheme": "openIdConnect",
        "issuer": "https://auth.example.com",
        "audience": "arp-agents",
        "claim_mapping": { "scope": "arp:scope", "permission": "arp:permission" }
      }
    ]
  }
}
```

This is the home for the A2A-auth federation referenced from
[Identity & Scopes](identity-and-scopes.md); control-plane (gRPC) OIDC is
configured separately and remains an optional extension there.

## Conformance

The core ARP conformance suite does not exercise this profile. Implementations
that ship a gateway SHOULD additionally verify:

- `proxy_url` is populated and points to `{gateway}/a2a/agents/{agent_id}`.
- The enriched `AgentCard` lists the direct interface first, the gateway second.
- Passthrough endpoints return the agent's A2A response **unmodified**.
- Scope/session denial returns HTTP 403 without forwarding.

The reference implementation demonstrates the enrichment side of this (the
`Gateway Profile (optional)` group in the Go conformance suite asserts
`proxy_url` population and interface ordering when a gateway base URL is
configured); the forwarding proxy itself is left to deployments.
