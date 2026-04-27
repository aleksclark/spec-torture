# Test Run: 766b12fc-58aa-4afe-accd-1782b7fd6312

**Spec:** a2a-v1  
**Runtime:** a2a-rs-helloworld   
**Started:** 2026-04-27 16:31:06  
**Completed:** 2026-04-27 16:31:06  

## Summary

| Metric | Count |
|--------|-------|
| Total | 20 |
| Passed | 17 |
| Failed | 3 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **85.0%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | PASS pass | 988.7µs |  |
| agent-card-content-type | PASS pass | 229.554µs |  |
| agent-card-optional-fields | PASS pass | 138.122µs |  |
| agent-card-skills | PASS pass | 142.269µs |  |
| agent-card-interfaces | PASS pass | 135.827µs |  |
| send-message-basic | PASS pass | 472.503µs |  |
| send-message-response-shape | PASS pass | 329.654µs |  |
| get-task | PASS pass | 444.601µs |  |
| get-task-not-found | PASS pass | 167.146µs |  |
| cancel-task | PASS pass | 485.218µs |  |
| message-text-part | PASS pass | 270.241µs |  |
| message-with-context | PASS pass | 586.039µs |  |
| optional-message-id | PASS pass | 273.357µs |  |
| streaming-send | PASS pass | 40.396405ms |  |
| streaming-events | PASS pass | 41.024403ms |  |
| jsonrpc-parse-error | FAIL fail | 218.694µs | expected JSON-RPC response but got none |
| jsonrpc-invalid-request | FAIL fail | 221.509µs | expected JSON-RPC response but got none |
| jsonrpc-method-not-found | PASS pass | 195.45µs |  |
| missing-required-params | FAIL fail | 378.947µs | JSON-RPC match failed: missing key at error |
| push-notification-not-supported | PASS pass | 729.329µs |  |
