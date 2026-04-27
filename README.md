# spec-torture

Torture-test agent runtimes against arbitrary protocol specifications.

## Overview

spec-torture is a Go + Docker test harness that validates whether agent runtimes correctly implement protocol specifications like MCP (Model Context Protocol), A2A, and others. It:

- Defines protocol specs as YAML test suites
- Launches runtimes in Docker containers
- Executes test cases (send/expect/assert steps) against the runtime
- Records results to SQLite and generates reports

## Quick Start

```bash
# Build
make build

# Validate a spec file
./bin/spec-torture validate specs/mcp/spec.yaml

# Run tests against a runtime
./bin/spec-torture run specs/mcp/spec.yaml --runtime my-agent --image my-agent:latest

# List loaded specs
./bin/spec-torture list

# View a test run report
./bin/spec-torture report <run-id>
./bin/spec-torture report <run-id> --format json
```

## Spec Format

Specs are YAML files defining test cases with send/expect steps:

```yaml
id: mcp-v1
name: Model Context Protocol
version: 0.1.0
transport: jsonrpc-stdio

test_cases:
  - id: initialize-handshake
    name: Initialize Handshake
    severity: required          # required | recommended | optional
    category: lifecycle
    timeout: 10s
    steps:
      - action: send
        payload:
          jsonrpc: "2.0"
          id: 1
          method: initialize
          params: { ... }
      - action: expect
        expect:
          jsonrpc: "2.0"
          id: 1
          result:
            protocolVersion: "*"    # wildcard matching
```

### Severity Levels (RFC 2119)

| Severity | Meaning |
|----------|---------|
| `required` | MUST implement — failure means non-compliant |
| `recommended` | SHOULD implement — failure is a warning |
| `optional` | MAY implement — informational only |

### Step Actions

| Action | Description |
|--------|-------------|
| `send` | Send a message to the runtime |
| `expect` | Wait for a response matching a pattern |
| `wait` | Pause for a duration |
| `assert` | Check a condition against accumulated state |
| `exec` | Run a command inside the container |

## Architecture

```
cmd/spec-torture/     CLI entry point (Cobra)
internal/
  schema/             Spec, TestCase, Step, TestResult, TestRun types
  runner/             Test execution engine + Docker lifecycle
  report/             JSON and Markdown report generation
  store/              SQLite persistence
specs/                Example spec definitions (YAML)
```

## Requirements

- Go 1.24+
- Docker (for running runtimes under test)

## License

MIT
