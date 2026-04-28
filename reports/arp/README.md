# ARP HTTP Compliance: awesometree Gap Analysis

**Spec:** `specs/arp/spec-http.yaml` (ARP v0.3 — HTTP endpoints only)  
**Runtime:** awesometree 0.1.0  
**Date:** 2026-04-28  
**Compliance:** 50.0% (12/24 tests passed)

## Summary

| Category | Passed | Failed | Total |
|----------|--------|--------|-------|
| a2a-proxy | 5 | 8 | 13 |
| a2a-proxy-routing | 2 | 2 | 4 |
| direct-vs-proxy | 1 | 2 | 3 |
| api-workspaces | 4 | 0 | 4 |
| **Total** | **12** | **12** | **24** |

## What Passes

### API / Management Plane (4/4 — 100%)

The existing REST management API is fully functional:

| Test | Status |
|------|--------|
| `api-workspaces-list` | PASS — `GET /api/workspaces` returns workspace list |
| `api-workspaces-get-single` | PASS — `GET /api/workspaces/arp-impl` returns correct workspace |
| `api-workspaces-get-nonexistent` | PASS — `GET /api/workspaces/nonexistent` returns 404 |
| `api-projects-list` | PASS — `GET /api/projects` returns project list |

### Error Handling — Nonexistent Agent (5/5 — 100%)

All "nonexistent agent" tests pass because the `/a2a/agents/*` path returns 404 universally
(the entire A2A proxy routing tree is unregistered):

| Test | Status |
|------|--------|
| `proxy-nonexistent-agent-404` | PASS — `GET /a2a/agents/nonexistent/.well-known/agent-card.json` → 404 |
| `proxy-nonexistent-agent-message-404` | PASS — `POST /a2a/agents/nonexistent/message:send` → 404 |
| `proxy-nonexistent-agent-task-404` | PASS — `GET /a2a/agents/nonexistent/tasks/...` → 404 |
| `proxy-nonexistent-agent-cancel-404` | PASS — `POST /a2a/agents/nonexistent/tasks/...:cancel` → 404 |
| `proxy-nonexistent-agent-stream-404` | PASS — `POST /a2a/agents/nonexistent/message:stream` → 404 |

### Routing Error Handling (2/2 — 100%)

| Test | Status |
|------|--------|
| `proxy-route-no-match-error` | PASS — `POST /a2a/route/message:send` with bad tags → 404 |
| `proxy-route-endpoint-exists` | PASS — routing endpoint returns 404 (no match), not 405 |

### Auth (1/1 pass — technically vacuous)

| Test | Status |
|------|--------|
| `proxy-enforces-token-scope` | PASS — invalid bearer token to `/a2a/agents/nonexistent/...` → 404 |

> **Note:** This passes because the entire `/a2a/agents` tree returns 404. Once the proxy is
> implemented, this test should be revisited to ensure it returns 401/403, not 404.

## What Fails

### Root Cause: A2A Proxy Not Implemented

**All 12 failures share one root cause:** the ARP A2A proxy endpoints (`/a2a/agents`,
`/a2a/discover`, `/a2a/route/message:send`) are not yet registered in the awesometree HTTP
router. Every request to these paths returns `404 Not Found`.

#### Discovery Endpoints (3 failures)

| Test | Expected | Got | Why |
|------|----------|-----|-----|
| `proxy-list-agents` | 200 | 404 | `GET /a2a/agents` — route not registered |
| `proxy-list-agents-content-type` | 200 + JSON | 404 | Same endpoint, checking content-type |
| `proxy-discover-endpoint` | 200 | 404 | `GET /a2a/discover` — route not registered |

**Fix:** Register `GET /a2a/agents` and `GET /a2a/discover` handlers that iterate over all
active workspaces, find agents with status=ready/busy, and return their enriched AgentCards.

#### Agent Card Enrichment (3 failures)

| Test | Expected | Got | Why |
|------|----------|-----|-----|
| `proxy-agent-card-well-known-chained` | 200 | 404 | Requires `/a2a/agents` to discover agent, then fetch card |
| `proxy-enriched-card-metadata-arp-chained` | 200 | 404 | Same chain — needs `metadata.arp` fields |
| `proxy-agent-card-interface-url-chained` | 200 | 404 | Same chain — checks `supportedInterfaces` rewriting |

**Fix:** Register `GET /a2a/agents/{id}/.well-known/agent-card.json` handler that:
1. Looks up the agent by ID
2. Fetches its native AgentCard from `direct_url/.well-known/agent-card.json`
3. Enriches with `metadata.arp` (agent_id, workspace, project, template, status, direct_url, started_at)
4. Rewrites `supportedInterfaces[].url` to point to the proxy URL

#### Message Proxying (2 failures)

| Test | Expected | Got | Why |
|------|----------|-----|-----|
| `proxy-send-message-chained` | 200 | 404 | `POST /a2a/agents/{id}/message:send` not registered |
| `proxy-send-streaming-message-chained` | 200 + SSE | 404 | `POST /a2a/agents/{id}/message:stream` not registered |

**Fix:** Register proxy handlers that forward A2A requests to the agent's `direct_url`:
- `POST /a2a/agents/{id}/message:send` → forward to `direct_url/message:send`
- `POST /a2a/agents/{id}/message:stream` → forward and stream SSE back

#### Routing (2 failures)

| Test | Expected | Got | Why |
|------|----------|-----|-----|
| `proxy-route-by-skill-tags` | 200 | 404 | `POST /a2a/route/message:send` returns 404 for all (no agents indexed) |
| `proxy-route-prefers-ready` | 200 | 404 | Same — skill-tag routing requires agent index |

**Fix:** Implement skill-based routing that:
1. Indexes all agent skills and tags
2. Matches incoming routing criteria against the index
3. Prefers agents with status=ready over busy
4. Forwards the request to the best-matching agent

#### Direct vs Proxy (2 failures)

| Test | Expected | Got | Why |
|------|----------|-----|-----|
| `proxy-no-auth-header-behavior` | 200 | 404 | `GET /a2a/agents` without auth header → 404 (endpoint doesn't exist) |
| `proxy-card-has-direct-url-chained` | 200 | 404 | Chain requires `/a2a/agents` to exist first |

**Fix:** These will pass automatically once the A2A proxy endpoints are registered.

## Implementation Priority

### Phase 1: Discovery (unblocks 8 tests)
1. `GET /a2a/agents` — list all ready/busy agents with enriched cards
2. `GET /a2a/discover` — same data, different shape if needed
3. `GET /a2a/agents/{id}/.well-known/agent-card.json` — enriched card with `metadata.arp`

### Phase 2: Message Proxying (unblocks 2 tests)
4. `POST /a2a/agents/{id}/message:send` — proxy to direct_url
5. `POST /a2a/agents/{id}/message:stream` — proxy SSE stream

### Phase 3: Task Proxying
6. `GET /a2a/agents/{id}/tasks/{tid}` — proxy GetTask
7. `POST /a2a/agents/{id}/tasks/{tid}:cancel` — proxy CancelTask

### Phase 4: Routing (unblocks 2 tests)
8. `POST /a2a/route/message:send` — skill-tag matching and routing

## Scope of This Test Suite

This suite tests **24 HTTP-based ARP test cases** out of 80 total in the full ARP spec.
The remaining **56 test cases** use MCP tool calls (`mcp_tool_call` action) which require
the `mcp-and-http` transport — not yet supported by spec-torture. Those tests cover:

- **mcp-project** (8 tests) — project/register, project/list, project/unregister
- **mcp-workspace** (11 tests) — workspace/create, workspace/list, workspace/get, workspace/destroy
- **mcp-agent-lifecycle** (15 tests) — agent/spawn, agent/list, agent/status, agent/stop, agent/restart
- **mcp-agent-messaging** (10 tests) — agent/message, agent/task, agent/task_status
- **auth-tokens** (12 tests) — token/create, scope narrowing, permission escalation

## Test Notes

- The "nonexistent agent" tests pass vacuously — the entire `/a2a` tree returns 404.
  Once the proxy is implemented, these should be re-evaluated to confirm they return 404
  specifically for _missing agents_ and not because the route doesn't exist.
- The `proxy-enforces-token-scope` test currently expects 404 (matching the non-existent
  endpoint). When auth is implemented, this should expect 401 or 403.
- The chained tests (suffix `-chained`) use `$prev.body.0.metadata.arp.agent_id` to extract
  agent IDs from the `/a2a/agents` listing. They cascade-fail when the first step returns 404.
