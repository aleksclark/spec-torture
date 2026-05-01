# ARP gRPC Compliance: localhost:19098

## ProjectService
  PASS  project-list                                            required
  PASS  project-list-response                                   required

## WorkspaceService
  PASS  workspace-list                                          required
  PASS  workspace-list-filter-project                           required
  PASS  workspace-get-nonexistent                               required
  PASS  workspace-destroy-nonexistent                           required

## AgentService
  PASS  agent-list                                              required
  PASS  agent-list-filter-workspace                             required
  PASS  agent-list-filter-status                                required
  PASS  agent-get-nonexistent                                   required
  PASS  agent-stop-nonexistent                                  required
  PASS  agent-restart-nonexistent                               required
  PASS  agent-message-nonexistent                               required
  PASS  agent-task-status-nonexistent                           required
  PASS  agent-spawn-missing-workspace                           required
  PASS  agent-spawn-missing-template                            required

## DiscoveryService
  PASS  discover-agents                                         required
  PASS  discover-local-scope                                    required
  PASS  discover-capability-filter                              required
  PASS  watch-agent-nonexistent                                 required

## TokenService
  PASS  token-create                                            required
  PASS  token-create-has-bearer                                 required
  FAIL  token-create-missing-subject                            required  expected InvalidArgument, got: 
  FAIL  token-create-missing-scope                              required  expected InvalidArgument, got: 

## Summary

| Metric | Count |
|--------|-------|
| Total | 24 |
| Passed | 22 |
| Failed | 2 |
| **Compliance** | **91.7%** |

## Failures

  FAIL  token-create-missing-subject: expected InvalidArgument, got: 
  FAIL  token-create-missing-scope: expected InvalidArgument, got: 
