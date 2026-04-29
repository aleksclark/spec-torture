# Test Run: 5c3cdc17-7bd6-4595-96f4-55d0572a5457

**Spec:** a2a-v1-client  
**Runtime:** crush-a2a-client   
**Started:** 2026-04-29 11:57:20  
**Completed:** 2026-04-29 12:00:16  

## Summary

| Metric | Count |
|--------|-------|
| Total | 20 |
| Passed | 18 |
| Failed | 2 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **90.0%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| client-discovery-fetches-agent-card | PASS pass | 4.945579924s |  |
| client-send-message-jsonrpc-envelope | PASS pass | 4.688646351s |  |
| client-send-message-method-name | PASS pass | 6.261314582s |  |
| client-send-message-content-type | PASS pass | 4.75015019s |  |
| client-send-message-has-message-id | PASS pass | 4.658387953s |  |
| client-send-message-has-role | PASS pass | 4.796941761s |  |
| client-send-message-has-parts | PASS pass | 5.251493315s |  |
| client-send-message-part-has-content | PASS pass | 6.901687424s |  |
| client-send-message-http-post | PASS pass | 4.640841549s |  |
| client-get-task-jsonrpc-envelope | PASS pass | 6.558275868s |  |
| client-get-task-method-name | FAIL fail | 1m0.032943553s | get task failed: crush run failed: signal: killed
stderr:  |
| client-get-task-has-id | PASS pass | 6.966634524s |  |
| client-send-message-context-propagation | FAIL fail | 15.124976762s | second message missing contextId (should propagate from first response) |
| client-handles-task-response | PASS pass | 6.536752575s |  |
| client-handles-message-response | PASS pass | 5.220235139s |  |
| client-handles-error-response | PASS pass | 4.698786023s |  |
| client-handles-task-not-found | PASS pass | 4.221900223s |  |
| client-streaming-method-name | PASS pass | 5.495643617s |  |
| client-streaming-accept-header | PASS pass | 5.603642326s |  |
| client-uses-v1-method-names | PASS pass | 9.098136256s |  |
