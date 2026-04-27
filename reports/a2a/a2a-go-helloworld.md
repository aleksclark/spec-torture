# Test Run: 55daf175-f2e5-4d18-a914-ffd4b1a0fe59

**Spec:** a2a-v1  
**Runtime:** a2a-go-helloworld   
**Started:** 2026-04-27 15:08:01  
**Completed:** 2026-04-27 15:08:01  

## Summary

| Metric | Count |
|--------|-------|
| Total | 19 |
| Passed | 9 |
| Failed | 10 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **47.4%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | FAIL fail | 861.63µs | body match failed: missing key at url |
| agent-card-content-type | PASS pass | 161.846µs |  |
| agent-card-optional-fields | PASS pass | 98.857µs |  |
| agent-card-skills | PASS pass | 65.644µs |  |
| send-message-basic | FAIL fail | 402.221µs | JSON-RPC match failed: missing key at result.id |
| send-message-creates-task | FAIL fail | 147.329µs | JSON-RPC match failed: missing key at result.kind |
| get-task | FAIL fail | 111.251µs | JSON-RPC match failed: missing key at result.id |
| get-task-not-found | PASS pass | 84.52µs |  |
| cancel-task | FAIL fail | 86.314µs | JSON-RPC match failed: missing key at result.id |
| message-text-part | FAIL fail | 88.908µs | JSON-RPC match failed: missing key at result.id |
| message-with-context | FAIL fail | 168.068µs | JSON-RPC match failed: missing key at result.id |
| streaming-send | PASS pass | 191.101µs |  |
| streaming-events | FAIL fail | 150.475µs | no SSE event matched: missing key at taskId |
| jsonrpc-parse-error | PASS pass | 79.611µs |  |
| jsonrpc-invalid-request | PASS pass | 65.374µs |  |
| jsonrpc-method-not-found | PASS pass | 59.192µs |  |
| missing-required-params | PASS pass | 62.298µs |  |
| optional-message-id | FAIL fail | 72.928µs | JSON-RPC match failed: missing key at result |
| push-notification-not-supported | FAIL fail | 86.774µs | JSON-RPC match failed: missing key at result.id |
