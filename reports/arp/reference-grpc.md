# ARP Reference Implementation — gRPC Conformance

## ProjectService
  PASS  project-list-ok                                      required
  PASS  project-list-contains-seed                           required
  PASS  project-register-requires-name                       required
  PASS  project-register-requires-repo                       required
  PASS  project-register-creates                             required
  PASS  project-register-appears-in-list                     required
  PASS  project-unregister-removes                           required
  PASS  project-unregister-nonexistent                       required
  PASS  project-unregister-active-agents                     required
  PASS  project-register-non-admin-denied                    required

## WorkspaceService
  PASS  workspace-create-requires-name                       required
  PASS  workspace-create-requires-project                    required
  PASS  workspace-create-unknown-project                     required
  PASS  workspace-create-ok-active                           required
  PASS  workspace-create-duplicate                           required
  PASS  workspace-list-ok                                    required
  PASS  workspace-list-filter-project                        required
  PASS  workspace-get-ok                                     required
  PASS  workspace-get-nonexistent                            required
  PASS  workspace-destroy-nonexistent                        required
  PASS  workspace-destroy-ok                                 required

## AgentService
  PASS  agent-spawn-missing-workspace                        required
  PASS  agent-spawn-missing-template                         required
  PASS  agent-spawn-unknown-workspace                        required
  PASS  agent-spawn-ok-ready                                 required
  PASS  agent-spawn-direct-url-format                        required
  PASS  agent-spawn-no-proxy-without-gateway                 required
  PASS  agent-list-ok                                        required
  PASS  agent-list-contains-seeded                           required
  PASS  agent-list-filter-workspace                          required
  PASS  agent-list-filter-status-ready                       required
  PASS  agent-list-filter-template                           required
  PASS  agent-get-ok                                         required
  PASS  agent-get-nonexistent                                required
  PASS  agent-stop-nonexistent                               required
  PASS  agent-stop-ok                                        required
  PASS  agent-restart-nonexistent                            required
  PASS  agent-restart-ok                                     required

## DiscoveryService
  PASS  discover-ok                                          required
  PASS  discover-local-scope                                 required
  PASS  discover-finds-crush                                 required
  PASS  discover-capability-crush                            required
  PASS  discover-capability-echo                             required
  PASS  discover-capability-nomatch                          required
  PASS  watch-agent-nonexistent                              required
  PASS  watch-agent-initial-and-stop                         required
  PASS  watch-workspace-initial-spawns                       recommended

## TokenService
  PASS  token-create-ok                                      required
  PASS  token-create-has-bearer                              required
  PASS  token-create-missing-subject                         required
  PASS  token-create-missing-scope                           required
  PASS  token-create-missing-permission                      required
  PASS  token-create-non-admin-denied                        required
  PASS  token-create-scope-widen-denied                      required

## Identity & Scopes
  PASS  scope-list-projects-filtered                         required
  PASS  scope-workspace-out-of-scope-denied                  required
  PASS  spawn-scope-widen-denied                             required
  PASS  spawn-perm-elevate-denied                            required
  PASS  spawn-scope-narrow-ok                                required
  PASS  session-isolation                                    required
  PASS  session-propagation                                  required
  PASS  project-perm-sees-all-in-project                     recommended

## Two-Interface & Enrichment
  PASS  enrich-no-proxy-without-gateway                      required
  PASS  enrich-direct-url                                    required
  PASS  enrich-card-arp-metadata                             required
  PASS  enrich-card-supported-interface-direct               required

## Gateway Profile (optional)
  PASS  gateway-proxy-url-populated                          recommended
  PASS  gateway-card-direct-still-primary                    recommended
  PASS  gateway-card-metadata-proxy-url                      recommended

## Summary

| Metric | Count |
|--------|-------|
| Total | 69 |
| Passed | 69 |
| Failed | 0 |
| **Compliance** | **100.0%** |
