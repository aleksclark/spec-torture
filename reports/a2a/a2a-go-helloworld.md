# Test Run: f82ae12e-9719-4fae-a4e1-6c2948bb1b93

**Spec:** a2a-v1  
**Runtime:** a2a-go-helloworld   
**Started:** 2026-04-27 16:31:06  
**Completed:** 2026-04-27 16:31:06  

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
| agent-card-well-known | PASS pass | 590.287µs |  |
| agent-card-content-type | PASS pass | 98.045µs |  |
| agent-card-optional-fields | PASS pass | 130.026µs |  |
| agent-card-skills | PASS pass | 74.05µs |  |
| agent-card-interfaces | PASS pass | 50.185µs |  |
| send-message-basic | PASS pass | 201.371µs |  |
| send-message-response-shape | PASS pass | 114.967µs |  |
| get-task | PASS pass | 165.072µs |  |
| get-task-not-found | PASS pass | 53.401µs |  |
| cancel-task | PASS pass | 173.999µs |  |
| message-text-part | PASS pass | 72.067µs |  |
| message-with-context | PASS pass | 141.768µs |  |
| optional-message-id | FAIL fail | 65.734µs | JSON-RPC match failed: missing key at result |
| streaming-send | PASS pass | 136.077µs |  |
| streaming-events | PASS pass | 183.818µs |  |
| jsonrpc-parse-error | PASS pass | 56.497µs |  |
| jsonrpc-invalid-request | PASS pass | 43.241µs |  |
| jsonrpc-method-not-found | PASS pass | 52.028µs |  |
| missing-required-params | PASS pass | 53.982µs |  |
| push-notification-not-supported | PASS pass | 129.225µs |  |
