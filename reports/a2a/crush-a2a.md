# Test Run: 5e69d660-20d7-4ea3-bfe6-86516b4b6a45

**Spec:** a2a-v1  
**Runtime:** crush-a2a   
**Started:** 2026-04-27 15:09:21  
**Completed:** 2026-04-27 15:09:21  

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
| agent-card-well-known | PASS pass | 727.846µs |  |
| agent-card-content-type | PASS pass | 150.835µs |  |
| agent-card-optional-fields | PASS pass | 81.594µs |  |
| agent-card-skills | PASS pass | 67.047µs |  |
| send-message-basic | FAIL fail | 3.074006ms | JSON-RPC match failed: missing key at result |
| send-message-creates-task | FAIL fail | 2.310782ms | JSON-RPC match failed: missing key at result |
| get-task | FAIL fail | 2.872825ms | JSON-RPC match failed: missing key at result |
| get-task-not-found | PASS pass | 98.917µs |  |
| cancel-task | FAIL fail | 2.284452ms | JSON-RPC match failed: missing key at result |
| message-text-part | FAIL fail | 2.293278ms | JSON-RPC match failed: missing key at result |
| message-with-context | FAIL fail | 2.064636ms | JSON-RPC match failed: missing key at result |
| streaming-send | FAIL fail | 1.942826ms | header content-type: expected "text/event-stream", got "application/json" |
| streaming-events | FAIL fail | 1.907118ms | expected SSE events but got none |
| jsonrpc-parse-error | PASS pass | 87.586µs |  |
| jsonrpc-invalid-request | PASS pass | 71.385µs |  |
| jsonrpc-method-not-found | PASS pass | 68.079µs |  |
| missing-required-params | PASS pass | 65.434µs |  |
| optional-message-id | FAIL fail | 2.576023ms | JSON-RPC match failed: missing key at result |
| push-notification-not-supported | FAIL fail | 2.26755ms | JSON-RPC match failed: missing key at result |
