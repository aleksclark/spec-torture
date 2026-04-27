# Test Run: 3acce350-f92f-4355-817d-4fece77b5361

**Spec:** a2a-v1  
**Runtime:** crush-a2a-native   
**Started:** 2026-04-27 13:25:24  
**Completed:** 2026-04-27 13:26:37  

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
| agent-card-well-known | PASS pass | 522.217µs |  |
| agent-card-content-type | PASS pass | 89.93µs |  |
| agent-card-optional-fields | PASS pass | 68.539µs |  |
| agent-card-skills | PASS pass | 81.815µs |  |
| send-message-basic | PASS pass | 9.248013918s |  |
| send-message-creates-task | PASS pass | 2.116115218s |  |
| get-task | PASS pass | 6.647572948s |  |
| get-task-not-found | PASS pass | 229.013µs |  |
| cancel-task | PASS pass | 14.390434492s |  |
| message-text-part | PASS pass | 1.835674249s |  |
| message-with-context | PASS pass | 4.182942425s |  |
| streaming-send | PASS pass | 12.263469226s |  |
| streaming-events | PASS pass | 14.63101635s |  |
| jsonrpc-parse-error | PASS pass | 569.267µs |  |
| jsonrpc-invalid-request | PASS pass | 872.54µs |  |
| jsonrpc-method-not-found | PASS pass | 111.141µs |  |
| missing-required-params | PASS pass | 74.08µs |  |
| missing-message-id | FAIL fail | 2.062952844s | JSON-RPC match failed: missing key at error |
| push-notification-not-supported | PASS pass | 5.44618477s |  |
