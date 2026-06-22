# ARP Reference Implementation (Go)

Reference **client and server libraries** for the [Agent Registry Protocol
(ARP) v0.5](../arp-spec/index.md), implemented in Go over gRPC. ARP is the
control plane that **creates, manages, and helps you discover** A2A agents;
A2A is how you talk to them. This package is a complete, spec-conformant
implementation of all five ARP gRPC services plus a client SDK.

The ARP/A2A boundary is the whole point: ARP returns each agent's `direct_url`
and never relays messages. Clients send A2A `SendMessage` / `GetTask` directly
to that URL with any A2A client. (A single authenticated A2A ingress is the
optional [gateway profile](../arp-spec/profile-a2a-gateway.md), not part of the
core gRPC services.)

## Layout

| Package | Role |
|---------|------|
| `gen/arp/v1` | Generated ARP protobuf + gRPC code (from `proto/arp/v1`, via `buf`). |
| `gen/lf/a2a/v1` | Generated A2A v1.0 types (from `proto/lf/a2a/v1/a2a.proto`): `AgentCard`, `AgentSkill`, `Task`, `Message`, `Part`, `Artifact`, etc. ARP references the typed `AgentCard` (not `google.protobuf.Struct`) wherever it returns agent cards. |
| `arp/server` | Reference server: implements `ProjectService`, `WorkspaceService`, `AgentService` (lifecycle only), `DiscoveryService`, `TokenService`. In-memory state, token scope/permission/session enforcement, process lifecycle state machine, direct-first AgentCard enrichment, server-streaming watches. `a2aconv.go` converts an agent's A2A HTTP+JSON AgentCard into the typed `lf.a2a.v1.AgentCard` via protojson. |
| `arp/client` | Reference client SDK: wraps the generated gRPC clients, injects bearer-token auth via gRPC metadata, and adds ergonomic helpers. |
| `arp/backend` | Pluggable agent process backend. `ExecBackend` launches real OS processes; `MockBackend` runs in-process A2A v1.0 agents for hermetic testing/seeding. |
| `arp/a2a` | Minimal A2A v1.0 HTTP+JSON client used by the server for **AgentCard discovery and health checks only** — no messaging (`SendMessage`/`GetTask` live in clients, not ARP). |
| `arp/conformance` | Executable conformance suite that drives the server through the client and verifies every normative requirement. |
| `cmd/arp-server` | Runnable reference server (gRPC + reflection; `-seed` installs demo fixtures). |
| `cmd/arp-conformance` | Runs the conformance suite and writes a markdown report. |

## Spec coverage

All RPCs from [arp-spec/](../arp-spec/) are implemented:

- **ProjectService** — `ListProjects`, `RegisterProject`, `UnregisterProject`
- **WorkspaceService** — `CreateWorkspace`, `ListWorkspaces`, `GetWorkspace`, `DestroyWorkspace`
- **AgentService** — `SpawnAgent`, `ListAgents`, `GetAgentStatus`, `StopAgent`, `RestartAgent` (lifecycle only — no messaging RPCs)
- **DiscoveryService** — `DiscoverAgents`, `WatchAgent` (stream), `WatchWorkspace` (stream)
- **TokenService** — `CreateToken`

Enforced behaviours include: the gRPC status-code tables in each
`services-*.md`; the `AgentStatus` process lifecycle state machine
(`starting → ready → stopping → stopped`, with `error`); child-token issuance
with monotonic scope-narrowing / permission-lowering; implicit session creation
and propagation; per-RPC scope/session visibility; and **direct-first** AgentCard
enrichment — `supported_interfaces[0]` is the agent's `direct_url` and the
`metadata.arp` block carries lifecycle context. `proxy_url` is populated only
when an optional gateway base URL is configured (the conformance suite covers
both the no-gateway default and the gateway-configured case).

Messaging is intentionally absent: clients reach agents by sending A2A directly
to `direct_url`. The reference `arp/a2a` client only fetches AgentCards and runs
health checks — it has no `SendMessage`/`GetTask`.

## Quick start

```bash
# 1. (Re)generate Go code from the protos — only needed if protos change.
make arp-tools     # installs protoc-gen-go + protoc-gen-go-grpc
make proto         # buf generate -> gen/arp/v1

# 2. Build the server and conformance tool.
make arp           # -> bin/arp-server, bin/arp-conformance

# 3. Run the reference server with demo fixtures (in-process mock agents).
./bin/arp-server -seed -addr :9099
#   seeds project "myapp", workspace "arp-test",
#   agents "echo-agent-001" and "crush-agent-001".

# 4. Talk to it with any gRPC client (reflection is enabled):
grpcurl -plaintext localhost:9099 list
grpcurl -plaintext localhost:9099 arp.v1.AgentService/ListAgents
```

### Using the client library

```go
cl, _ := client.Dial("localhost:9099") // localhost => admin via localhost_admin
proj, _ := cl.RegisterProject(ctx, "myapp", "/src/myapp", template)
ws, _   := cl.CreateWorkspace(ctx, "feat-auth", "myapp")
agent, _ := cl.SpawnAgent(ctx, "feat-auth", "crush", "coder")

// ARP located the agent; now talk A2A directly — ARP is not in the path.
directURL := agent.GetDirectUrl() // or cl.DirectURL(ctx, agent.GetId())
// POST {directURL}/message:send  with any A2A client ...

// Scoped, non-admin caller:
tok, _ := cl.CreateToken(ctx, "lead", &arpv1.Scope{Projects: []string{"myapp"}}, arpv1.Permission_PERMISSION_PROJECT, 0)
scoped, _ := client.Dial("localhost:9099", client.WithToken(tok.GetBearerToken()))
```

## Conformance

The suite boots the server with the in-process mock backend and exercises
every service end-to-end (including the agent lifecycle, scope/session
enforcement, and streaming watches). It is the authoritative validation that
the implementation conforms to the spec.

```bash
make arp-test                       # go test ./arp/...
./bin/arp-conformance -out reports/arp/reference-grpc.md
make arp-run                        # boot seeded server + run both suites
```

Two reports are produced (both 100% compliant):

- [`reports/arp/reference-grpc.md`](../reports/arp/reference-grpc.md) — the Go
  conformance suite (69 checks across all five services + identity/scopes +
  direct-first enrichment + the optional gateway profile).
- [`reports/arp/reference-grpcurl.md`](../reports/arp/reference-grpcurl.md) —
  the repository's existing gRPC harness
  (`agents/arp-reference/run-grpc.sh`, a `grpcurl`/reflection variant of
  `agents/awesometree/run-grpc.sh`) run against the reference server. It passes
  40/40, including the two `TokenService` validation cases the awesometree
  daemon fails.

## Backends

`arp/server` is backend-agnostic. Provide a `backend.Backend` in `server.Config`:

- `backend.NewExecBackend()` — spawns the template's `command` as a local
  process, injecting the allocated port via `port_env` and the child token via
  `ARP_TOKEN`, then polls the health-check path until ready. This is the
  default for `arp-server` (without `-seed`).
- `backend.NewMockBackend()` — serves a deterministic in-process A2A v1.0 agent
  per spawn; used by the conformance suite and `arp-server -seed`.
