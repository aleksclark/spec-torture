# AGENTS.md

## Commands

```bash
make build                    # Build binary to bin/spec-torture
make test                     # go test ./...
make lint                     # go vet ./...
make tidy                     # go mod tidy
make clean                    # Remove bin/ and spec-torture.db
make docker                   # Build Docker image

# Task runner (preferred for running torture suites)
task build                    # Build with source/output caching
task validate                 # Validate all spec YAML files
task torture:all              # Run A2A suite against all agents
task torture:a2a-go           # Run against Go SDK helloworld
task torture:a2a-rs           # Run against Rust SDK helloworld
task clean                    # Remove bin/ and all agents/.build/
```

## Architecture

spec-torture is a **protocol conformance test harness** for AI agent runtimes. It reads declarative test suites from YAML spec files, executes send/expect step sequences against live endpoints (HTTP or Docker), evaluates responses with partial matching, persists results to SQLite, and generates reports.

### Data Flow

```
spec.yaml → loadSpec() → schema.Spec
  → runner.Run() iterates TestCases
    → each TestCase: create Transport → execute Steps sequentially
      → send step: Transport.Send() → stores TransportResponse
      → expect step: matchExpect() against last TransportResponse
      → assert step: evaluateAssert() with condition parsing
    → aggregate into TestRun with Summary
  → store.SaveTestRun() → SQLite
  → report.Write() → stdout (markdown or JSON)
```

### Key Packages

| Package | Role |
|---------|------|
| `cmd/spec-torture` | CLI entry point (Cobra). Commands: `run`, `validate`, `list`, `report` |
| `internal/schema` | Data types: `Spec`, `TestCase`, `Step`, `TestResult`, `TestRun`, `Summary`. Validation logic. No external dependencies beyond stdlib. |
| `internal/runner` | Test execution engine. Routes to transport based on spec transport type and CLI flags. |
| `internal/runner/transport.go` | `Transport` interface (`Send`, `LastResponse`), variable interpolation (`$prev.field.path`), response matching (`matchExpect`), JSON path resolution |
| `internal/runner/http_runner.go` | `HTTPTransport` — plain HTTP REST. Supports polling via `poll` config in payloads. |
| `internal/runner/jsonrpc_runner.go` | `JSONRPCTransport` — JSON-RPC 2.0 over HTTP POST (used by A2A spec). Handles SSE streaming, raw body sends, auto-detects request type from payload keys. |
| `internal/runner/evaluator.go` | Partial deep-match: only keys present in `expected` must exist in `actual`. `"*"` wildcard matches any non-nil value. Numeric comparison handles int/float64/json.Number mismatches. |
| `internal/runner/docker.go` | `DockerManager` — container lifecycle for stdio-transport specs. Only initialized when `--url` is not provided. |
| `internal/store` | SQLite persistence (modernc.org/sqlite, pure Go). Auto-migrates on open. WAL mode. |
| `internal/report` | Renders `TestRun` as JSON or Markdown to any `io.Writer`. |

### Transport Routing

The runner selects transport based on two factors:

1. **`--url` flag present** → HTTP mode (no Docker). Transport chosen by `spec.Transport`:
   - `jsonrpc-http` → `JSONRPCTransport` (posts to `baseURL + rpcPath`)
   - `http-rest` → `HTTPTransport` (direct HTTP requests)
2. **`--image` flag present** → Docker mode. Container started per test case. `executeStepLegacy` is a stub — Docker-based step execution is not yet implemented.

### Agent Runner Scripts

Each agent under `agents/` has a `run.sh` that:
1. Clones/updates the upstream SDK repo into `.build/`
2. Builds the agent binary
3. Starts it on a fixed port
4. Runs `spec-torture run` twice (markdown + JSON output)
5. Cleans up on exit via trap

Port assignments: Go=8201, Rust=8202. RPC paths differ per SDK (`/invoke` for Go, `/jsonrpc` for Rust).

## Spec YAML Format

Specs define test suites. Key structural rules:

- `transport` must be one of: `jsonrpc-stdio`, `jsonrpc-http`, `http-rest`, `grpc`
- `severity` must be: `required`, `recommended`, `optional` (maps to RFC 2119)
- Test case IDs must be unique within a spec
- Steps execute sequentially; first non-pass step aborts the test case
- `timeout` on test cases uses Go duration strings (`10s`, `60s`, `120s`)

### Step Actions and Payload Conventions

**`send` step** — payload keys determine the request type:
- `http_method` + `path` → plain HTTP request (used by both transports)
- `jsonrpc` + `method` → JSON-RPC request (JSONRPCTransport only)
- `raw_body` → raw string body sent as-is (for malformed input testing)
- `headers` in payload → custom HTTP headers on the request
- `body` → JSON-marshaled request body
- `poll` → enables polling: `{interval: "2s", max_attempts: 15}`, terminates on `status: completed|failed|cancelled`

**`expect` step** — keys checked against last `TransportResponse`:
- `http_status` → exact HTTP status code match
- `headers` → case-insensitive partial match; `"*"` matches any value
- `body` → partial deep match against JSON body
- `jsonrpc` → partial deep match against JSON-RPC response
- `sse_event` → match against any SSE event in the stream; `"*"` means "at least one event"

**`assert` step** — evaluates conditions:
- `condition: "field.path in ['val1', 'val2']"` — set membership check
- Only `in [...]` conditions are implemented

**`wait` step** — `payload.duration` as Go duration string

### Variable Interpolation

`$prev.field.path` references resolve against the last `TransportResponse`. The resolution order in `mergeResponseData`: `JSONRPCResp` → `Body` → last SSE event. Used for chaining steps (e.g., send message, then get task by `$prev.result.id`).

## Gotchas

### Response Matching

- **Partial matching only**: expect steps check that expected keys exist in the response — extra keys in the actual response are ignored. This is intentional for forward-compatibility testing.
- **`"*"` wildcard**: matches any non-nil value including empty strings, empty arrays, nested objects. It does NOT match a missing key.
- **Header matching is case-insensitive**: headers are normalized to lowercase. The expect value is checked with `strings.Contains` (substring match), not exact equality. Exception: `"*"` skips the check entirely.
- **Numeric comparison**: the evaluator handles int/float64/json.Number cross-type comparison (within 1e-9 epsilon). YAML integers may deserialize as `int` or `float64` depending on context.

### JSONRPCTransport Request Type Detection

`JSONRPCTransport.Send()` auto-detects the request type by checking payload keys in this order:
1. `http_method` present → plain HTTP (e.g., agent card discovery)
2. `raw_body` present → raw POST (e.g., malformed JSON tests)
3. `jsonrpc` present → JSON-RPC request

If none match, it returns an error. When writing spec YAML, ensure at least one of these keys is present in every send step.

### Streaming Detection

`JSONRPCTransport` hard-codes streaming for methods: `SendStreamingMessage`, `SubscribeToTask`, `message/stream`, `tasks/resubscribe`. These are sent with `Accept: text/event-stream` and parsed as SSE. Any other method gets a standard JSON-RPC response. Adding a new streaming method requires updating the condition in `sendJSONRPC()`.

### SSE Parsing

SSE events are read with a 10-second hard-coded timeout. The parser expects `data: <json>` lines separated by blank lines. Only JSON-parseable data lines are captured; non-JSON events are silently dropped.

### Docker Transport is Stubbed

`executeStepLegacy` always returns `StatusError` with "step execution not yet implemented for Docker transport". The MCP spec (`jsonrpc-stdio`) cannot actually execute tests — it validates YAML structure only.

### SQLite Database

A `spec-torture.db` file is created in the working directory on every `run` command. It auto-migrates schema on open. The `clean` make target removes it. The database is used for `list` and `report` commands to recall past runs.

### Agent `.build/` Directories

Agent runner scripts clone upstream repos into `agents/<name>/.build/`. These are gitignored. First run of `task torture:a2a-rs` takes ~60s for Rust compilation. Subsequent runs use cached artifacts. `task clean` removes all `.build/` dirs.

### go.mod Version

The module declares `go 1.25.0` which requires a tip/pre-release Go toolchain. Standard Go 1.24.x will reject this. Ensure your Go version matches.

## Report Output

Reports go to `reports/<protocol>/` as both `.md` and `.json`. The runner outputs markdown to stdout by default; agent `run.sh` scripts redirect to files. Log output (slog) goes to stderr and is captured in `.log` files.

Compliance percentage = `passed / (total - skipped) * 100`. Skipped tests (tag-filtered) don't count against compliance.
