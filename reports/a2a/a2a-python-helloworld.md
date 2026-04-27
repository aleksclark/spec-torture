# Test Run: 1a072720-0017-4e34-982d-a2b0d44e0aec

**Spec:** a2a-v1  
**Runtime:** a2a-python-helloworld   
**Started:** 2026-04-27 13:42:39  
**Completed:** 2026-04-27 13:42:39  

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
| agent-card-well-known | PASS pass | 1.351364ms |  |
| agent-card-content-type | PASS pass | 333.651µs |  |
| agent-card-optional-fields | PASS pass | 245.995µs |  |
| agent-card-skills | PASS pass | 228.382µs |  |
| send-message-basic | FAIL fail | 2.146377ms | JSON-RPC match failed: missing key at result |
| send-message-creates-task | FAIL fail | 1.264861ms | JSON-RPC match failed: missing key at result |
| get-task | FAIL fail | 1.218853ms | JSON-RPC match failed: missing key at result |
| get-task-not-found | PASS pass | 384.907µs |  |
| cancel-task | FAIL fail | 1.263539ms | JSON-RPC match failed: missing key at result |
| message-text-part | FAIL fail | 1.208194ms | JSON-RPC match failed: missing key at result |
| message-with-context | FAIL fail | 1.18494ms | JSON-RPC match failed: missing key at result |
| streaming-send | FAIL fail | 1.21155ms | header content-type: expected "text/event-stream", got "application/json" |
| streaming-events | FAIL fail | 1.188266ms | expected SSE events but got none |
| jsonrpc-parse-error | PASS pass | 677.581µs |  |
| jsonrpc-invalid-request | PASS pass | 446.724µs |  |
| jsonrpc-method-not-found | PASS pass | 239.402µs |  |
| missing-required-params | FAIL fail | 264.54µs | JSON-RPC match failed: mismatch at error.code: expected -32602 (int), got -32009... |
| optional-message-id | FAIL fail | 1.219625ms | JSON-RPC match failed: missing key at result |
| push-notification-not-supported | FAIL fail | 1.223964ms | JSON-RPC match failed: missing key at result |
