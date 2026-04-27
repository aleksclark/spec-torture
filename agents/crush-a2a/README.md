# crush-a2a

A2A v1.0 frontend for [Crush](https://github.com/charmbracelet/crush) — translates
A2A JSON-RPC calls to Crush's native HTTP REST protocol.

**Source:** <https://github.com/aleksclark/crush-a2a>

## Prerequisites

- Go 1.24+
- `crush` CLI installed (`go install github.com/charmbracelet/crush@latest` or from source)

## How it works

The `run.sh` script:

1. Starts `crush server` (native Crush backend on unix socket)
2. Builds and starts `crush-a2a` (A2A v1.0 frontend, port 8200)
3. Runs the spec-torture A2A suite
4. Kills both processes on exit

## Manual run

```bash
# Start crush server
crush server &

# Build and run crush-a2a
cd .build/crush-a2a
go build -o crush-a2a ./cmd/crush-a2a
./crush-a2a --crush-addr unix:///tmp/crush-$(id -u).sock \
  --workspace-path /tmp/crush-a2a-test --port 8200

# In another terminal
curl http://localhost:8200/.well-known/agent-card.json
```
