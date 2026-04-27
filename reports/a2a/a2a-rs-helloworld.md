# Test Run: 66f0b4ca-7862-4832-aaf0-9cf94d390d94

**Spec:** a2a-v1  
**Runtime:** a2a-rs-helloworld   
**Started:** 2026-04-27 16:50:58  
**Completed:** 2026-04-27 16:50:59  

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
| agent-card-well-known | PASS pass | 795.692µs |  |
| agent-card-content-type | PASS pass | 193.385µs |  |
| agent-card-optional-fields | PASS pass | 151.476µs |  |
| agent-card-skills | PASS pass | 129.654µs |  |
| agent-card-interfaces | PASS pass | 129.074µs |  |
| send-message-basic | PASS pass | 580.616µs |  |
| send-message-response-shape | PASS pass | 307.09µs |  |
| get-task | PASS pass | 479.275µs |  |
| get-task-not-found | PASS pass | 162.176µs |  |
| cancel-task | PASS pass | 443.257µs |  |
| message-text-part | PASS pass | 267.435µs |  |
| message-with-context | PASS pass | 549.968µs |  |
| optional-message-id | PASS pass | 268.377µs |  |
| streaming-send | PASS pass | 41.485146ms |  |
| streaming-events | PASS pass | 41.025919ms |  |
| jsonrpc-parse-error | FAIL fail | 422.888µs | expected JSON-RPC response but got none |
| jsonrpc-invalid-request | FAIL fail | 157.036µs | expected JSON-RPC response but got none |
| jsonrpc-method-not-found | PASS pass | 175.572µs |  |
| missing-required-params | FAIL fail | 381.511µs | JSON-RPC match failed: missing key at error |
| push-notification-not-supported | PASS pass | 532.976µs |  |
