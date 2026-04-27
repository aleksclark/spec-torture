# Test Run: 280613f9-828b-458d-bc9b-3db9aae67bcd

**Spec:** a2a-v1  
**Runtime:** a2a-js-sample   
**Started:** 2026-04-27 13:45:46  
**Completed:** 2026-04-27 13:45:46  

## Summary

| Metric | Count |
|--------|-------|
| Total | 19 |
| Passed | 8 |
| Failed | 11 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **42.1%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | PASS pass | 1.309766ms |  |
| agent-card-content-type | PASS pass | 406.458µs |  |
| agent-card-optional-fields | PASS pass | 326.017µs |  |
| agent-card-skills | PASS pass | 316.147µs |  |
| send-message-basic | FAIL fail | 5.116221ms | JSON-RPC match failed: missing key at result |
| send-message-creates-task | FAIL fail | 624.912µs | JSON-RPC match failed: missing key at result |
| get-task | FAIL fail | 405.818µs | JSON-RPC match failed: missing key at result |
| get-task-not-found | PASS pass | 901.184µs |  |
| cancel-task | FAIL fail | 446.914µs | JSON-RPC match failed: missing key at result |
| message-text-part | FAIL fail | 316.719µs | JSON-RPC match failed: missing key at result |
| message-with-context | FAIL fail | 341.436µs | JSON-RPC match failed: missing key at result |
| streaming-send | FAIL fail | 307.24µs | header content-type: expected "text/event-stream", got "application/json; charse... |
| streaming-events | FAIL fail | 601.788µs | expected SSE events but got none |
| jsonrpc-parse-error | PASS pass | 770.436µs |  |
| jsonrpc-invalid-request | PASS pass | 304.846µs |  |
| jsonrpc-method-not-found | PASS pass | 283.376µs |  |
| missing-required-params | FAIL fail | 261.965µs | JSON-RPC match failed: mismatch at error.code: expected -32602 (int), got -32601... |
| optional-message-id | FAIL fail | 560.89µs | JSON-RPC match failed: missing key at result |
| push-notification-not-supported | FAIL fail | 256.144µs | JSON-RPC match failed: missing key at result |
