# Test Run: a1eeaf47-51ad-4c3f-a646-1fb2faa60b18

**Spec:** arp-http-v0.3  
**Runtime:** awesometree 0.1.0  
**Started:** 2026-04-29 15:02:36  
**Completed:** 2026-04-29 15:02:36  

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
| openapi-spec-endpoint | PASS pass | 773.25µs |  |
| api-workspaces-list | PASS pass | 213.934µs |  |
| api-workspaces-list-content-type | PASS pass | 144.092µs |  |
| api-workspaces-get-nonexistent | PASS pass | 73.7µs |  |
| api-projects-list | PASS pass | 198.925µs |  |
| api-projects-get-nonexistent | PASS pass | 53.952µs |  |
| proxy-list-agents | PASS pass | 53.651µs |  |
| proxy-discover-endpoint | PASS pass | 52.158µs |  |
| proxy-discover-filter-workspace | PASS pass | 46.578µs |  |
| proxy-discover-filter-capability | PASS pass | 41.509µs |  |
| proxy-nonexistent-agent-card-404 | PASS pass | 45.246µs |  |
| proxy-nonexistent-agent-message-404 | PASS pass | 66.355µs |  |
| proxy-nonexistent-agent-task-404 | PASS pass | 45.957µs |  |
| proxy-nonexistent-agent-cancel-404 | PASS pass | 53.02µs |  |
| proxy-nonexistent-agent-stream-404 | PASS pass | 47.229µs |  |
| proxy-route-endpoint-accepts-post | PASS pass | 67.257µs |  |
| proxy-route-no-match-404 | PASS pass | 67.077µs |  |
| proxy-list-then-card | FAIL fail | 93.005µs | expected HTTP status 200, got 404 |
| proxy-card-interface-points-to-proxy | FAIL fail | 99.498µs | expected HTTP status 200, got 404 |
| proxy-send-message | FAIL fail | 101.111µs | expected HTTP status 200, got 404 |
| proxy-route-by-skill-tags | FAIL fail | 97.033µs | expected HTTP status 200, got 404 |
