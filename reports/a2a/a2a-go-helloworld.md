# Test Run: a72de98b-4a54-404d-8858-3809c434c1e8

**Spec:** a2a-v1  
**Runtime:** a2a-go-helloworld   
**Started:** 2026-04-27 16:18:25  
**Completed:** 2026-04-27 16:18:25  

## Summary

| Metric | Count |
|--------|-------|
| Total | 20 |
| Passed | 18 |
| Failed | 2 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **90.0%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | PASS pass | 659.408µs |  |
| agent-card-content-type | PASS pass | 101.803µs |  |
| agent-card-optional-fields | PASS pass | 59.743µs |  |
| agent-card-skills | PASS pass | 49.794µs |  |
| agent-card-interfaces | PASS pass | 61.556µs |  |
| send-message-basic | PASS pass | 414.414µs |  |
| send-message-response-shape | PASS pass | 124.485µs |  |
| get-task | PASS pass | 175.301µs |  |
| get-task-not-found | PASS pass | 83.759µs |  |
| cancel-task | PASS pass | 196.792µs |  |
| message-text-part | PASS pass | 83.598µs |  |
| message-with-context | PASS pass | 205.258µs |  |
| optional-message-id | FAIL fail | 69.311µs | JSON-RPC match failed: missing key at result |
| streaming-send | PASS pass | 183.898µs |  |
| streaming-events | FAIL fail | 259.281µs | expected sse_event must be a map |
| jsonrpc-parse-error | PASS pass | 183.487µs |  |
| jsonrpc-invalid-request | PASS pass | 133.433µs |  |
| jsonrpc-method-not-found | PASS pass | 85.271µs |  |
| missing-required-params | PASS pass | 65.665µs |  |
| push-notification-not-supported | PASS pass | 212.693µs |  |
