# ARP gRPC Compliance: localhost:19098

## ProjectService
  FAIL  project-list                                            required  Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  project-list-response                                   required  expected output to contain 'projects'

## WorkspaceService
  FAIL  workspace-list                                          required  Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  workspace-list-filter-project                           required  Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  workspace-get-nonexistent                               required  expected NotFound, got: 
  FAIL  workspace-destroy-nonexistent                           required  expected NotFound, got: 

## AgentService
  FAIL  agent-list                                              required  Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  agent-list-filter-workspace                             required  Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  agent-list-filter-status                                required  Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  agent-get-nonexistent                                   required  expected NotFound, got: 
  FAIL  agent-stop-nonexistent                                  required  expected NotFound, got: 
  FAIL  agent-restart-nonexistent                               required  expected NotFound, got: 
  FAIL  agent-message-nonexistent                               required  expected NotFound, got: 
  FAIL  agent-task-status-nonexistent                           required  expected NotFound, got: 
  FAIL  agent-spawn-missing-workspace                           required  expected InvalidArgument, got: 
  FAIL  agent-spawn-missing-template                            required  expected InvalidArgument, got: 

## DiscoveryService
  FAIL  discover-agents                                         required  Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  discover-local-scope                                    required  Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  discover-capability-filter                              required  Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  watch-agent-nonexistent                                 required  expected NotFound, got: 

## TokenService
  FAIL  token-create                                            required  Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  token-create-has-bearer                                 required  expected output to contain 'bearerToken'
  FAIL  token-create-missing-subject                            required  expected InvalidArgument, got: 
  FAIL  token-create-missing-scope                              required  expected InvalidArgument, got: 

## Summary

| Metric | Count |
|--------|-------|
| Total | 24 |
| Passed | 0 |
| Failed | 24 |
| **Compliance** | **0.0%** |

## Failures

  FAIL  project-list: Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  project-list-response: expected output to contain 'projects'
  FAIL  workspace-list: Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  workspace-list-filter-project: Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  workspace-get-nonexistent: expected NotFound, got: 
  FAIL  workspace-destroy-nonexistent: expected NotFound, got: 
  FAIL  agent-list: Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  agent-list-filter-workspace: Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  agent-list-filter-status: Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  agent-get-nonexistent: expected NotFound, got: 
  FAIL  agent-stop-nonexistent: expected NotFound, got: 
  FAIL  agent-restart-nonexistent: expected NotFound, got: 
  FAIL  agent-message-nonexistent: expected NotFound, got: 
  FAIL  agent-task-status-nonexistent: expected NotFound, got: 
  FAIL  agent-spawn-missing-workspace: expected InvalidArgument, got: 
  FAIL  agent-spawn-missing-template: expected InvalidArgument, got: 
  FAIL  discover-agents: Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  discover-local-scope: Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  discover-capability-filter: Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  watch-agent-nonexistent: expected NotFound, got: 
  FAIL  token-create: Failed to dial target host "localhost:19098": context deadline exceeded
  FAIL  token-create-has-bearer: expected output to contain 'bearerToken'
  FAIL  token-create-missing-subject: expected InvalidArgument, got: 
  FAIL  token-create-missing-scope: expected InvalidArgument, got: 
