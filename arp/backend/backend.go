// Package backend defines the agent process backend abstraction used by the
// ARP reference server. A Backend is responsible for starting an agent so that
// it serves A2A v1.0 HTTP+JSON on a local port, and for stopping it again.
//
// Two backends are provided:
//
//   - ExecBackend launches a real OS process from an AgentTemplate command.
//   - MockBackend runs an in-process A2A v1.0 agent, used for hermetic
//     conformance testing and for seeding demo servers.
package backend

import (
	"context"
	"time"

	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// SpawnSpec describes a single agent instance the server wants started.
type SpawnSpec struct {
	// AgentID is the server-assigned unique instance id.
	AgentID string
	// Template is the resolved agent template (command, env, health check).
	Template *arpv1.AgentTemplate
	// Workspace is the parent workspace name.
	Workspace string
	// WorkspaceDir is the absolute working directory for the process.
	WorkspaceDir string
	// Port is the port the server allocated for this agent.
	Port int
	// Env holds additional per-instance environment variables.
	Env map[string]string
	// Token is the child ARP token injected as ARP_TOKEN.
	Token string
}

// Handle is a running agent the server can address and stop.
type Handle interface {
	// DirectURL is the base URL of the agent's A2A v1.0 endpoint.
	DirectURL() string
	// Port is the port the agent listens on.
	Port() int
	// PID is the OS process id, or 0 for in-process/remote backends.
	PID() int
	// Stop terminates the agent, waiting up to graceMs before forcing.
	Stop(ctx context.Context, graceMs int) error
}

// Backend starts agents on behalf of the server.
type Backend interface {
	// Spawn starts an agent and returns once it is reachable (health passing)
	// or returns an error if it never became ready.
	Spawn(ctx context.Context, spec SpawnSpec) (Handle, error)
}

// HealthDefaults returns the effective health-check parameters for a template,
// applying spec defaults for any unset fields.
func HealthDefaults(t *arpv1.AgentTemplate) (path string, interval, timeout time.Duration, retries int) {
	path = "/.well-known/agent-card.json"
	interval = 200 * time.Millisecond
	timeout = 3 * time.Second
	retries = 25
	if t == nil || t.GetHealthCheck() == nil {
		return path, interval, timeout, retries
	}
	hc := t.GetHealthCheck()
	if hc.GetPath() != "" {
		path = hc.GetPath()
	}
	if hc.GetIntervalMs() > 0 {
		interval = time.Duration(hc.GetIntervalMs()) * time.Millisecond
	}
	if hc.GetTimeoutMs() > 0 {
		timeout = time.Duration(hc.GetTimeoutMs()) * time.Millisecond
	}
	if hc.GetRetries() > 0 {
		retries = int(hc.GetRetries())
	}
	return path, interval, timeout, retries
}
