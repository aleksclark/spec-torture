# Test Run: 8b569b4b-a2a8-420f-9d3b-43efbd29c407

**Spec:** arp-http-v0.3  
**Runtime:** awesometree 0.1.0  
**Started:** 2026-04-28 10:51:51  
**Completed:** 2026-04-28 10:51:51  

## Summary

| Metric | Count |
|--------|-------|
| Total | 24 |
| Passed | 12 |
| Failed | 12 |
| Errors | 0 |
| Skipped | 0 |
| Timeouts | 0 |
| **Compliance** | **50.0%** |

## Results

| Test Case | Status | Duration | Error |
|-----------|--------|----------|-------|
| proxy-list-agents | FAIL fail | 429.462µs | expected HTTP status 200, got 404 |
| proxy-list-agents-content-type | FAIL fail | 82.396µs | expected HTTP status 200, got 404 |
| proxy-discover-endpoint | FAIL fail | 50.806µs | expected HTTP status 200, got 404 |
| proxy-nonexistent-agent-404 | PASS pass | 37.03µs |  |
| proxy-nonexistent-agent-message-404 | PASS pass | 48.361µs |  |
| proxy-nonexistent-agent-task-404 | PASS pass | 35.977µs |  |
| proxy-nonexistent-agent-cancel-404 | PASS pass | 32.02µs |  |
| proxy-nonexistent-agent-stream-404 | PASS pass | 40.667µs |  |
| proxy-route-no-match-error | PASS pass | 38.072µs |  |
| proxy-route-endpoint-exists | PASS pass | 35.768µs |  |
| proxy-enforces-token-scope | PASS pass | 33.774µs |  |
| proxy-no-auth-header-behavior | FAIL fail | 29.356µs | expected HTTP status 200, got 404 |
| api-workspaces-list | PASS pass | 182.625µs |  |
| api-workspaces-get-single | PASS pass | 70.723µs |  |
| api-workspaces-get-nonexistent | PASS pass | 68.93µs |  |
| api-projects-list | PASS pass | 176.985µs |  |
| proxy-agent-card-well-known-chained | FAIL fail | 36.358µs | expected HTTP status 200, got 404 |
| proxy-enriched-card-metadata-arp-chained | FAIL fail | 33.443µs | expected HTTP status 200, got 404 |
| proxy-agent-card-interface-url-chained | FAIL fail | 30.348µs | expected HTTP status 200, got 404 |
| proxy-send-message-chained | FAIL fail | 38.202µs | expected HTTP status 200, got 404 |
| proxy-send-streaming-message-chained | FAIL fail | 30.157µs | expected HTTP status 200, got 404 |
| proxy-card-has-direct-url-chained | FAIL fail | 29.847µs | expected HTTP status 200, got 404 |
| proxy-route-by-skill-tags | FAIL fail | 37.702µs | expected HTTP status 200, got 404 |
| proxy-route-prefers-ready | FAIL fail | 34.265µs | expected HTTP status 200, got 404 |
