# Test Run: b7cd6389-5beb-47bb-9b1d-0fd69eb0ab77

**Spec:** a2a-v1  
**Runtime:** a2a-go-helloworld   
**Started:** 2026-04-27 16:50:57  
**Completed:** 2026-04-27 16:50:57  

## Summary

| Metric | Count |
|--------|-------|
| Total | 20 |
| Passed | 19 |
| Failed | 1 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **95.0%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | PASS pass | 494.233µs |  |
| agent-card-content-type | PASS pass | 119.626µs |  |
| agent-card-optional-fields | PASS pass | 70.463µs |  |
| agent-card-skills | PASS pass | 64.893µs |  |
| agent-card-interfaces | PASS pass | 57.889µs |  |
| send-message-basic | PASS pass | 222.139µs |  |
| send-message-response-shape | PASS pass | 113.013µs |  |
| get-task | PASS pass | 168.327µs |  |
| get-task-not-found | PASS pass | 58.701µs |  |
| cancel-task | PASS pass | 228.531µs |  |
| message-text-part | PASS pass | 124.825µs |  |
| message-with-context | PASS pass | 213.493µs |  |
| optional-message-id | FAIL fail | 80.432µs | JSON-RPC match failed: missing key at result |
| streaming-send | PASS pass | 189.938µs |  |
| streaming-events | PASS pass | 150.684µs |  |
| jsonrpc-parse-error | PASS pass | 86.694µs |  |
| jsonrpc-invalid-request | PASS pass | 66.185µs |  |
| jsonrpc-method-not-found | PASS pass | 65.122µs |  |
| missing-required-params | PASS pass | 71.575µs |  |
| push-notification-not-supported | PASS pass | 205.247µs |  |
