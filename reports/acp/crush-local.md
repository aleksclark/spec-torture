# Test Run: 9c612c00-bb0b-462a-b401-4d72e7cd4456

**Spec:** acp-v1  
**Runtime:** crush-local   
**Started:** 2026-04-27 10:20:13  
**Completed:** 2026-04-27 10:20:43  

## Summary

| Metric | Count |
|--------|-------|
| Total | 34 |
| Passed | 30 |
| Failed | 4 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **88.2%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| ping | PASS pass | 370.761µs |  |
| ping-content-type | PASS pass | 107.624µs |  |
| agents-list | PASS pass | 113.113µs |  |
| agent-manifest-fields | PASS pass | 92.875µs |  |
| agent-by-name | PASS pass | 85.822µs |  |
| agent-not-found | PASS pass | 71.905µs |  |
| create-run-sync | PASS pass | 2.03641312s |  |
| run-has-required-fields | PASS pass | 1.954191353s |  |
| run-output-has-agent-message | PASS pass | 3.065152158s |  |
| run-message-has-parts | PASS pass | 2.441903311s |  |
| create-run-async | PASS pass | 167.176µs |  |
| poll-run-status | PASS pass | 182.244µs |  |
| poll-until-complete | PASS pass | 1.001061649s |  |
| run-completed-has-output | FAIL fail | 1.000513084s | body match failed: expected at least 1 items at output, got 0 |
| create-run-stream | PASS pass | 2.067987ms |  |
| stream-has-run-created | PASS pass | 684.603µs |  |
| stream-has-message-parts | FAIL fail | 645.921µs | NDJSON stream does not contain event matching map[part:map[content:*] type:messa... |
| stream-has-run-completed | PASS pass | 740.318µs |  |
| stream-x-run-id-header | PASS pass | 531.104µs |  |
| stream-content-type | PASS pass | 561.541µs |  |
| empty-input-rejected | PASS pass | 77.797µs |  |
| wrong-agent-name | PASS pass | 66.706µs |  |
| empty-content-rejected | PASS pass | 56.006µs |  |
| invalid-json-rejected | PASS pass | 69.151µs |  |
| cancel-active-run | FAIL fail | 129.805µs | expected HTTP 200, got HTTP 202 |
| cancel-completed-run | PASS pass | 763.713µs |  |
| cancel-nonexistent-run | PASS pass | 52.259µs |  |
| session-persistence | PASS pass | 5.96906807s |  |
| session-export | PASS pass | 1.932260456s |  |
| session-import | PASS pass | 2.206362644s |  |
| events-list | PASS pass | 2.07128766s |  |
| events-stream-ndjson | PASS pass | 2.146653709s |  |
| session-message-events | FAIL fail | 2.182222586s | NDJSON stream does not contain event matching map[message:map[role:*] type:sessi... |
| session-snapshot-event | PASS pass | 2.434462124s |  |
