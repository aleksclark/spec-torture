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

## Lifecycle (echo-agent)
## Requires running echo-agent-001 in arp-test workspace
  PASS  lifecycle-get-echo-agent                                required
  PASS  lifecycle-echo-agent-status-ready                       required
  PASS  lifecycle-echo-agent-has-template                       required
  PASS  lifecycle-echo-agent-has-workspace                      required
  PASS  lifecycle-echo-agent-has-port                           required
  PASS  lifecycle-list-shows-echo                               required
  PASS  lifecycle-list-filter-ready                             required
  PASS  lifecycle-list-filter-workspace                         required
  PASS  lifecycle-send-message-echo-error-propagation           required
  PASS  lifecycle-discover-finds-echo                           required
  PASS  lifecycle-discover-by-echo-capability                   required
  PASS  lifecycle-discover-no-match                             required
  PASS  lifecycle-workspace-has-agents                          required
  PASS  lifecycle-workspace-is-active                           required

## Lifecycle (crush-agent)
## Requires running crush-agent-001 in arp-test workspace
  PASS  lifecycle-crush-agent-ready                             required
  PASS  lifecycle-send-message-crush                            required
  PASS  lifecycle-crush-message-has-result                      required
  PASS  lifecycle-crush-message-blocking                        required
  PASS  lifecycle-discover-finds-crush                          required
  PASS  lifecycle-discover-by-crush-capability                  required

## TokenService
  PASS  token-create                                            required
  PASS  token-create-has-bearer                                 required
  FAIL  token-create-missing-subject                            required  expected InvalidArgument, got: 
  FAIL  token-create-missing-scope                              required  expected InvalidArgument, got: 

## Summary

| Metric | Count |
|--------|-------|
| Total | 44 |
| Passed | 42 |
| Failed | 2 |
| **Compliance** | **95.5%** |

## Failures

  FAIL  token-create-missing-subject: expected InvalidArgument, got: 
  FAIL  token-create-missing-scope: expected InvalidArgument, got: 
