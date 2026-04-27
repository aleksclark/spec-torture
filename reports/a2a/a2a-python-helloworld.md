# Test Run: a1baeba3-4d18-4e1e-a38c-ef14bf4a1a30

**Spec:** a2a-v1  
**Runtime:** a2a-python-helloworld   
**Started:** 2026-04-27 10:26:57  
**Completed:** 2026-04-27 10:26:57  

## Summary

| Metric | Count |
|--------|-------|
| Total | 19 |
| Passed | 9 |
| Failed | 10 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **47.4%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | PASS pass | 834.868µs |  |
| agent-card-content-type | PASS pass | 278.536µs |  |
| agent-card-optional-fields | PASS pass | 251.866µs |  |
| agent-card-skills | PASS pass | 306.889µs |  |
| send-message-basic | FAIL fail | 309.484µs | JSON-RPC match failed: missing key at result |
| send-message-creates-task | FAIL fail | 228.461µs | JSON-RPC match failed: missing key at result |
| get-task | FAIL fail | 212.081µs | JSON-RPC match failed: missing key at result |
| get-task-not-found | PASS pass | 202.903µs |  |
| cancel-task | FAIL fail | 197.153µs | JSON-RPC match failed: missing key at result |
| message-text-part | FAIL fail | 195.89µs | JSON-RPC match failed: missing key at result |
| message-with-context | FAIL fail | 198.074µs | JSON-RPC match failed: missing key at result |
| streaming-send | FAIL fail | 186.823µs | header content-type: expected "text/event-stream", got "application/json" |
| streaming-events | FAIL fail | 193.095µs | expected SSE events but got none |
| jsonrpc-parse-error | PASS pass | 580.486µs |  |
| jsonrpc-invalid-request | PASS pass | 401.549µs |  |
| jsonrpc-method-not-found | PASS pass | 213.203µs |  |
| missing-required-params | FAIL fail | 204.817µs | JSON-RPC match failed: mismatch at error.code: expected -32602 (int), got -32601... |
| missing-message-id | PASS pass | 211.84µs |  |
| push-notification-not-supported | FAIL fail | 271.082µs | JSON-RPC match failed: missing key at result |
