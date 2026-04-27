# Test Run: 7cd8eb9e-7a2b-49b5-81b1-653b8b452cf7

**Spec:** a2a-v1  
**Runtime:** a2a-js-sample   
**Started:** 2026-04-27 10:26:17  
**Completed:** 2026-04-27 10:26:27  

## Summary

| Metric | Count |
|--------|-------|
| Total | 19 |
| Passed | 18 |
| Failed | 1 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **94.7%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | PASS pass | 1.057157ms |  |
| agent-card-content-type | PASS pass | 207.221µs |  |
| agent-card-optional-fields | PASS pass | 178.086µs |  |
| agent-card-skills | PASS pass | 161.676µs |  |
| send-message-basic | PASS pass | 1.000991059s |  |
| send-message-creates-task | PASS pass | 1.001954158s |  |
| get-task | PASS pass | 1.001134831s |  |
| get-task-not-found | PASS pass | 311.349µs |  |
| cancel-task | PASS pass | 1.002090547s |  |
| message-text-part | PASS pass | 1.000982063s |  |
| message-with-context | PASS pass | 2.002084955s |  |
| streaming-send | PASS pass | 1.000864641s |  |
| streaming-events | PASS pass | 1.000835948s |  |
| jsonrpc-parse-error | PASS pass | 637.103µs |  |
| jsonrpc-invalid-request | PASS pass | 457.364µs |  |
| jsonrpc-method-not-found | PASS pass | 461.442µs |  |
| missing-required-params | FAIL fail | 437.767µs | JSON-RPC match failed: mismatch at error.code: expected -32602 (int), got -32603... |
| missing-message-id | PASS pass | 337.147µs |  |
| push-notification-not-supported | PASS pass | 1.001288283s |  |
