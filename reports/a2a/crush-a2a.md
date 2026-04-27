# Test Run: 112d90e2-da03-4127-ab1c-cb38697a87a5

**Spec:** a2a-v1  
**Runtime:** crush-a2a-native   
**Started:** 2026-04-27 12:50:02  
**Completed:** 2026-04-27 12:51:08  

## Summary

| Metric | Count |
|--------|-------|
| Total | 19 |
| Passed | 17 |
| Failed | 2 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **89.5%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | PASS pass | 787.811µs |  |
| agent-card-content-type | PASS pass | 111.992µs |  |
| agent-card-optional-fields | PASS pass | 78.419µs |  |
| agent-card-skills | PASS pass | 67.698µs |  |
| send-message-basic | PASS pass | 8.957211162s |  |
| send-message-creates-task | PASS pass | 1.888119979s |  |
| get-task | FAIL fail | 7.454482171s | JSON-RPC match failed: missing key at result |
| get-task-not-found | PASS pass | 94.849µs |  |
| cancel-task | PASS pass | 7.438368503s |  |
| message-text-part | PASS pass | 2.246268876s |  |
| message-with-context | PASS pass | 4.039629435s |  |
| streaming-send | PASS pass | 11.961188775s |  |
| streaming-events | PASS pass | 14.725432824s |  |
| jsonrpc-parse-error | PASS pass | 1.191454ms |  |
| jsonrpc-invalid-request | PASS pass | 408.342µs |  |
| jsonrpc-method-not-found | PASS pass | 128.574µs |  |
| missing-required-params | PASS pass | 112.442µs |  |
| missing-message-id | FAIL fail | 2.791464272s | JSON-RPC match failed: missing key at error |
| push-notification-not-supported | PASS pass | 4.39378359s |  |
