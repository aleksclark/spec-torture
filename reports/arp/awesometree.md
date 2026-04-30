# Test Run: 909ee580-7ea4-40ad-927e-48f292e7b183

**Spec:** arp-http-v0.3  
**Runtime:** awesometree 0.2.0  
**Started:** 2026-04-30 15:59:32  
**Completed:** 2026-04-30 15:59:32  

## Summary

| Metric | Count |
|--------|-------|
| Total | 21 |
| Passed | 17 |
| Failed | 4 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **81.0%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| openapi-spec-endpoint | PASS pass | 3.428079ms |  |
| api-workspaces-list | PASS pass | 201.68µs |  |
| api-workspaces-list-content-type | PASS pass | 162.427µs |  |
| api-workspaces-get-nonexistent | PASS pass | 82.767µs |  |
| api-projects-list | PASS pass | 225.936µs |  |
| api-projects-get-nonexistent | PASS pass | 49.633µs |  |
| proxy-list-agents | PASS pass | 56.998µs |  |
| proxy-discover-endpoint | PASS pass | 67.838µs |  |
| proxy-discover-filter-workspace | PASS pass | 62.348µs |  |
| proxy-discover-filter-capability | PASS pass | 44.564µs |  |
| proxy-nonexistent-agent-card-404 | PASS pass | 59.772µs |  |
| proxy-nonexistent-agent-message-404 | PASS pass | 74.941µs |  |
| proxy-nonexistent-agent-task-404 | PASS pass | 63.28µs |  |
| proxy-nonexistent-agent-cancel-404 | PASS pass | 48.642µs |  |
| proxy-nonexistent-agent-stream-404 | PASS pass | 56.978µs |  |
| proxy-route-endpoint-accepts-post | PASS pass | 76.074µs |  |
| proxy-route-no-match-404 | PASS pass | 64.842µs |  |
| proxy-list-then-card | FAIL fail | 127.962µs | expected HTTP status 200, got 404 |
| proxy-card-interface-points-to-proxy | FAIL fail | 101.681µs | expected HTTP status 200, got 404 |
| proxy-send-message | FAIL fail | 114.807µs | expected HTTP status 200, got 404 |
| proxy-route-by-skill-tags | FAIL fail | 103.285µs | expected HTTP status 200, got 404 |
