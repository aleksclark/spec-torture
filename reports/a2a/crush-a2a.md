# Test Run: 3f52c842-666f-4fe9-af89-36924346d3ec

**Spec:** a2a-v1  
**Runtime:** crush-a2a-v2   
**Started:** 2026-04-27 12:03:25  
**Completed:** 2026-04-27 12:04:09  

## Summary

| Metric | Count |
|--------|-------|
| Total | 19 |
| Passed | 16 |
| Failed | 3 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **84.2%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | PASS pass | 649.447µs |  |
| agent-card-content-type | PASS pass | 146.908µs |  |
| agent-card-optional-fields | PASS pass | 80.302µs |  |
| agent-card-skills | PASS pass | 61.737µs |  |
| send-message-basic | PASS pass | 6.527146793s |  |
| send-message-creates-task | PASS pass | 4.908172177s |  |
| get-task | FAIL fail | 1.35957172s | JSON-RPC match failed: missing key at result |
| get-task-not-found | PASS pass | 103.756µs |  |
| cancel-task | PASS pass | 14.607005512s |  |
| message-text-part | PASS pass | 2.13342438s |  |
| message-with-context | PASS pass | 3.60864949s |  |
| streaming-send | PASS pass | 2.558047945s |  |
| streaming-events | PASS pass | 5.008951617s |  |
| jsonrpc-parse-error | PASS pass | 193.476µs |  |
| jsonrpc-invalid-request | PASS pass | 92.485µs |  |
| jsonrpc-method-not-found | PASS pass | 87.446µs |  |
| missing-required-params | FAIL fail | 330.224µs | JSON-RPC match failed: mismatch at error.code: expected -32602 (int), got -32603... |
| missing-message-id | FAIL fail | 2.277939061s | JSON-RPC match failed: missing key at error |
| push-notification-not-supported | PASS pass | 1.092183735s |  |
