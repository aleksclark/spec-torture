# Test Run: 0bd79267-5071-4e5d-aecf-73ffd38e031c

**Spec:** a2a-v1  
**Runtime:** a2a-rs-helloworld   
**Started:** 2026-04-27 15:08:01  
**Completed:** 2026-04-27 15:08:01  

## Summary

| Metric | Count |
|--------|-------|
| Total | 19 |
| Passed | 5 |
| Failed | 14 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **26.3%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | FAIL fail | 1.364051ms | body match failed: missing key at url |
| agent-card-content-type | PASS pass | 257.287µs |  |
| agent-card-optional-fields | PASS pass | 176.534µs |  |
| agent-card-skills | PASS pass | 219.395µs |  |
| send-message-basic | FAIL fail | 316.789µs | JSON-RPC match failed: missing key at result |
| send-message-creates-task | FAIL fail | 190.971µs | JSON-RPC match failed: missing key at result |
| get-task | FAIL fail | 213.995µs | JSON-RPC match failed: missing key at result |
| get-task-not-found | PASS pass | 167.837µs |  |
| cancel-task | FAIL fail | 160.383µs | JSON-RPC match failed: missing key at result |
| message-text-part | FAIL fail | 161.756µs | JSON-RPC match failed: missing key at result |
| message-with-context | FAIL fail | 171.725µs | JSON-RPC match failed: missing key at result |
| streaming-send | FAIL fail | 162.768µs | header content-type: expected "text/event-stream", got "application/json" |
| streaming-events | FAIL fail | 180.852µs | expected SSE events but got none |
| jsonrpc-parse-error | FAIL fail | 143.682µs | expected JSON-RPC response but got none |
| jsonrpc-invalid-request | FAIL fail | 120.317µs | expected JSON-RPC response but got none |
| jsonrpc-method-not-found | PASS pass | 141.057µs |  |
| missing-required-params | FAIL fail | 430.104µs | JSON-RPC match failed: missing key at error |
| optional-message-id | FAIL fail | 167.227µs | JSON-RPC match failed: missing key at result |
| push-notification-not-supported | FAIL fail | 156.646µs | JSON-RPC match failed: missing key at result |
