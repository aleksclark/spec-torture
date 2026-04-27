# Test Run: 8f9ae9cd-bed5-415d-a1b3-e4ba3a7e9731

**Spec:** a2a-v1  
**Runtime:** crush-a2a   
**Started:** 2026-04-27 16:17:20  
**Completed:** 2026-04-27 16:17:20  

## Summary

| Metric | Count |
|--------|-------|
| Total | 20 |
| Passed | 8 |
| Failed | 12 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **40.0%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | FAIL fail | 690.577µs | body match failed: missing key at supportedInterfaces |
| agent-card-content-type | PASS pass | 76.445µs |  |
| agent-card-optional-fields | PASS pass | 42.249µs |  |
| agent-card-skills | PASS pass | 58.01µs |  |
| agent-card-interfaces | FAIL fail | 81.234µs | body match failed: missing key at supportedInterfaces |
| send-message-basic | FAIL fail | 133.062µs | JSON-RPC match failed: missing key at result |
| send-message-response-shape | FAIL fail | 65.534µs | JSON-RPC match failed: missing key at result |
| get-task | FAIL fail | 94.75µs | JSON-RPC match failed: missing key at result |
| get-task-not-found | PASS pass | 74.131µs |  |
| cancel-task | FAIL fail | 50.084µs | JSON-RPC match failed: missing key at result |
| message-text-part | FAIL fail | 58.671µs | JSON-RPC match failed: missing key at result |
| message-with-context | FAIL fail | 75.934µs | JSON-RPC match failed: missing key at result |
| optional-message-id | FAIL fail | 133.923µs | JSON-RPC match failed: missing key at result |
| streaming-send | FAIL fail | 63.551µs | header content-type: expected "text/event-stream", got "application/json" |
| streaming-events | FAIL fail | 48.602µs | expected sse_event must be a map |
| jsonrpc-parse-error | PASS pass | 56.066µs |  |
| jsonrpc-invalid-request | PASS pass | 47.52µs |  |
| jsonrpc-method-not-found | PASS pass | 44.404µs |  |
| missing-required-params | PASS pass | 42.33µs |  |
| push-notification-not-supported | FAIL fail | 60.324µs | JSON-RPC match failed: missing key at result |
