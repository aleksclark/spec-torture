# a2a-rs (Official Rust SDK Helloworld)

The official A2A Rust SDK helloworld agent — a simple echo agent that
returns the user's message prefixed with "Echo: ".

**Source:** <https://github.com/a2aproject/a2a-rs>  
**Path:** `examples/helloworld/`

## Prerequisites

- Rust toolchain (cargo) — any recent stable version

## How it works

The `run.sh` script:

1. Clones the `a2a-rs` repo (shallow)
2. Builds the helloworld example with `cargo build` (first run takes ~60s)
3. Starts the agent on port 8202
4. Runs the spec-torture A2A suite with `--rpc-path /jsonrpc`
5. Kills the server on exit

## Notes

- JSON-RPC endpoint is at `/jsonrpc` (not root `/`)
- REST endpoint at `/rest`
- Agent card at `/.well-known/agent-card.json`
- Returns `Task` (not `Message`) from `SendMessage`
- No LLM dependency — pure echo, instant responses
- First build is slow (compiling all dependencies); subsequent runs use cached artifacts
