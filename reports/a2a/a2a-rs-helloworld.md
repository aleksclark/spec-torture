# Test Run: d6ba8c19-8ed4-4e54-a8d1-e51f97e4c9e4

**Spec:** a2a-v1  
**Runtime:** a2a-rs-helloworld   
**Started:** 2026-04-27 16:18:25  
**Completed:** 2026-04-27 16:18:25  

## Summary

| Metric | Count |
|--------|-------|
| Total | 20 |
| Passed | 16 |
| Failed | 4 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **80.0%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| agent-card-well-known | PASS pass | 1.037804ms |  |
| agent-card-content-type | PASS pass | 187.425µs |  |
| agent-card-optional-fields | PASS pass | 219.455µs |  |
| agent-card-skills | PASS pass | 122.252µs |  |
| agent-card-interfaces | PASS pass | 129.885µs |  |
| send-message-basic | PASS pass | 543.439µs |  |
| send-message-response-shape | PASS pass | 311.83µs |  |
| get-task | PASS pass | 423.501µs |  |
| get-task-not-found | PASS pass | 161.566µs |  |
| cancel-task | PASS pass | 460.11µs |  |
| message-text-part | PASS pass | 274.649µs |  |
| message-with-context | PASS pass | 578.785µs |  |
| optional-message-id | PASS pass | 268.849µs |  |
| streaming-send | PASS pass | 40.485238ms |  |
| streaming-events | FAIL fail | 40.971839ms | expected sse_event must be a map |
| jsonrpc-parse-error | FAIL fail | 179.83µs | expected JSON-RPC response but got none |
| jsonrpc-invalid-request | FAIL fail | 158.409µs | expected JSON-RPC response but got none |
| jsonrpc-method-not-found | PASS pass | 158.129µs |  |
| missing-required-params | FAIL fail | 280.701µs | JSON-RPC match failed: missing key at error |
| push-notification-not-supported | PASS pass | 472.955µs |  |
