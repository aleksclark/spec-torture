# a2a-go (Official Go SDK Helloworld)

The official A2A Go SDK v2.x helloworld agent — a simple echo agent that
returns "Hello, world!" for every message.

**Source:** <https://github.com/a2aproject/a2a-go>  
**Path:** `examples/helloworld/server/jsonrpc/`

## Prerequisites

- Go 1.24+

## How it works

The `run.sh` script:

1. Clones the `a2a-go` repo (shallow)
2. Builds and starts the helloworld JSON-RPC server on port 8201
3. Runs the spec-torture A2A suite with `--rpc-path /invoke`
4. Kills the server on exit

## Notes

- JSON-RPC endpoint is at `/invoke` (not root `/`)
- Agent card at `/.well-known/agent-card.json`
- Returns `Message` (not `Task`) from `SendMessage` — valid per v1.0 proto oneof
- No LLM dependency — pure echo, instant responses
