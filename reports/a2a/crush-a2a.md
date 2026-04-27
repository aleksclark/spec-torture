# Test Run: 0f25ff50-8400-44d4-bc5c-8fce589df62b

**Spec:** a2a-v1  
**Runtime:** crush-a2a   
**Started:** 2026-04-27 11:35:08  
**Completed:** 2026-04-27 11:35:08  

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
| agent-card-well-known | PASS pass | 560.609µs |  |
| agent-card-content-type | PASS pass | 122.712µs |  |
| agent-card-optional-fields | PASS pass | 83.207µs |  |
| agent-card-skills | PASS pass | 63.159µs |  |
| send-message-basic | FAIL fail | 118.725µs | JSON-RPC match failed: missing key at result |
| send-message-creates-task | FAIL fail | 77.136µs | JSON-RPC match failed: missing key at result |
| get-task | FAIL fail | 79.069µs | JSON-RPC match failed: missing key at result |
| get-task-not-found | PASS pass | 83.979µs |  |
| cancel-task | FAIL fail | 60.334µs | JSON-RPC match failed: missing key at result |
| message-text-part | FAIL fail | 83.187µs | JSON-RPC match failed: missing key at result |
| message-with-context | FAIL fail | 64.371µs | JSON-RPC match failed: missing key at result |
| streaming-send | FAIL fail | 58.54µs | header content-type: expected "text/event-stream", got "application/json" |
| streaming-events | FAIL fail | 80.402µs | expected SSE events but got none |
| jsonrpc-parse-error | PASS pass | 52.229µs |  |
| jsonrpc-invalid-request | PASS pass | 63.049µs |  |
| jsonrpc-method-not-found | PASS pass | 72.307µs |  |
| missing-required-params | FAIL fail | 72.137µs | JSON-RPC match failed: mismatch at error.code: expected -32602 (int), got -32601... |
| missing-message-id | PASS pass | 78.178µs |  |
| push-notification-not-supported | FAIL fail | 52.8µs | JSON-RPC match failed: missing key at result |
