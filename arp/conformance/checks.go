package conformance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/aleksclark/spec-torture/arp/client"
	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
	a2av1 "github.com/aleksclark/spec-torture/gen/lf/a2a/v1"
)

type run struct {
	env     *Env
	admin   *client.Client
	results []Result
}

// Run executes the full conformance suite against env and returns results.
func Run(env *Env) ([]Result, error) {
	admin, err := env.Admin()
	if err != nil {
		return nil, err
	}
	defer admin.Close()
	r := &run{env: env, admin: admin}

	r.projectChecks()
	r.workspaceChecks()
	r.agentChecks()
	r.discoveryChecks()
	r.tokenChecks()
	r.identityChecks()
	r.enrichmentChecks()
	r.gatewayChecks()
	return r.results, nil
}

func (r *run) check(group, id, sev string, fn func() (bool, string)) {
	pass, detail := fn()
	r.results = append(r.results, Result{Group: group, ID: id, Severity: sev, Pass: pass, Detail: detail})
}

// ---------------------------------------------------------------------------
// ProjectService
// ---------------------------------------------------------------------------

func (r *run) projectChecks() {
	const g = "ProjectService"
	ctx, cancel := shortCtx()
	defer cancel()

	r.check(g, "project-list-ok", Required, func() (bool, string) {
		_, err := r.admin.ListProjects(ctx)
		return ok(err)
	})
	r.check(g, "project-list-contains-seed", Required, func() (bool, string) {
		ps, err := r.admin.ListProjects(ctx)
		if err != nil {
			return false, err.Error()
		}
		return truthy(containsProject(ps, "myapp"), "expected project myapp in list")
	})
	r.check(g, "project-register-requires-name", Required, func() (bool, string) {
		_, err := r.admin.Project.RegisterProject(ctx, &arpv1.RegisterProjectRequest{Repo: "/tmp/x"})
		return wantCode(err, codes.InvalidArgument)
	})
	r.check(g, "project-register-requires-repo", Required, func() (bool, string) {
		_, err := r.admin.Project.RegisterProject(ctx, &arpv1.RegisterProjectRequest{Name: "needs-repo"})
		return wantCode(err, codes.InvalidArgument)
	})
	r.check(g, "project-register-creates", Required, func() (bool, string) {
		p, err := r.admin.RegisterProject(ctx, "proj-reg", "/tmp/proj-reg")
		if err != nil {
			return false, err.Error()
		}
		if p.GetName() != "proj-reg" || p.GetRepo() != "/tmp/proj-reg" {
			return false, fmt.Sprintf("unexpected project %v", p)
		}
		return truthy(p.GetBranch() == "main", "expected default branch main")
	})
	r.check(g, "project-register-appears-in-list", Required, func() (bool, string) {
		ps, err := r.admin.ListProjects(ctx)
		if err != nil {
			return false, err.Error()
		}
		return truthy(containsProject(ps, "proj-reg"), "registered project missing from list")
	})
	r.check(g, "project-unregister-removes", Required, func() (bool, string) {
		if _, err := r.admin.RegisterProject(ctx, "proj-del", "/tmp/proj-del"); err != nil {
			return false, err.Error()
		}
		if err := r.admin.UnregisterProject(ctx, "proj-del"); err != nil {
			return false, err.Error()
		}
		ps, err := r.admin.ListProjects(ctx)
		if err != nil {
			return false, err.Error()
		}
		return truthy(!containsProject(ps, "proj-del"), "unregistered project still listed")
	})
	r.check(g, "project-unregister-nonexistent", Required, func() (bool, string) {
		err := r.admin.UnregisterProject(ctx, "nonexistent-00000")
		return wantCode(err, codes.NotFound)
	})
	r.check(g, "project-unregister-active-agents", Required, func() (bool, string) {
		err := r.admin.UnregisterProject(ctx, "myapp")
		return wantCode(err, codes.FailedPrecondition)
	})
	r.check(g, "project-register-non-admin-denied", Required, func() (bool, string) {
		tc, cl, err := r.tokenClient(ctx, "proj-tok", scopeProjects("myapp"), arpv1.Permission_PERMISSION_PROJECT)
		if err != nil {
			return false, err.Error()
		}
		defer cl.Close()
		_, err = tc.Project.RegisterProject(ctx, &arpv1.RegisterProjectRequest{Name: "nope", Repo: "/tmp/nope"})
		return wantCode(err, codes.PermissionDenied)
	})
}

// ---------------------------------------------------------------------------
// WorkspaceService
// ---------------------------------------------------------------------------

func (r *run) workspaceChecks() {
	const g = "WorkspaceService"
	ctx, cancel := shortCtx()
	defer cancel()

	r.check(g, "workspace-create-requires-name", Required, func() (bool, string) {
		_, err := r.admin.Workspace.CreateWorkspace(ctx, &arpv1.CreateWorkspaceRequest{Project: "myapp"})
		return wantCode(err, codes.InvalidArgument)
	})
	r.check(g, "workspace-create-requires-project", Required, func() (bool, string) {
		_, err := r.admin.Workspace.CreateWorkspace(ctx, &arpv1.CreateWorkspaceRequest{Name: "ws-noproj"})
		return wantCode(err, codes.InvalidArgument)
	})
	r.check(g, "workspace-create-unknown-project", Required, func() (bool, string) {
		_, err := r.admin.CreateWorkspace(ctx, "ws-unknownproj", "nonexistent-00000")
		return wantCode(err, codes.NotFound)
	})
	r.check(g, "workspace-create-ok-active", Required, func() (bool, string) {
		w, err := r.admin.CreateWorkspace(ctx, "ws-a", "myapp")
		if err != nil {
			return false, err.Error()
		}
		return truthy(w.GetStatus() == arpv1.WorkspaceStatus_WORKSPACE_STATUS_ACTIVE, "workspace not ACTIVE")
	})
	r.check(g, "workspace-create-duplicate", Required, func() (bool, string) {
		_, err := r.admin.CreateWorkspace(ctx, "ws-a", "myapp")
		return wantCode(err, codes.AlreadyExists)
	})
	r.check(g, "workspace-list-ok", Required, func() (bool, string) {
		_, err := r.admin.ListWorkspaces(ctx)
		return ok(err)
	})
	r.check(g, "workspace-list-filter-project", Required, func() (bool, string) {
		resp, err := r.admin.Workspace.ListWorkspaces(ctx, &arpv1.ListWorkspacesRequest{Project: "myapp"})
		if err != nil {
			return false, err.Error()
		}
		return truthy(containsWorkspace(resp.GetWorkspaces(), "arp-test"), "arp-test missing from project filter")
	})
	r.check(g, "workspace-get-ok", Required, func() (bool, string) {
		w, err := r.admin.GetWorkspace(ctx, "arp-test")
		if err != nil {
			return false, err.Error()
		}
		if w.GetProject() != "myapp" {
			return false, "wrong project"
		}
		return truthy(len(w.GetAgents()) >= 2, "expected seeded agents in workspace")
	})
	r.check(g, "workspace-get-nonexistent", Required, func() (bool, string) {
		_, err := r.admin.GetWorkspace(ctx, "nonexistent-ws-00000")
		return wantCode(err, codes.NotFound)
	})
	r.check(g, "workspace-destroy-nonexistent", Required, func() (bool, string) {
		err := r.admin.DestroyWorkspace(ctx, "nonexistent-ws-00000")
		return wantCode(err, codes.NotFound)
	})
	r.check(g, "workspace-destroy-ok", Required, func() (bool, string) {
		if _, err := r.admin.CreateWorkspace(ctx, "ws-del", "myapp"); err != nil {
			return false, err.Error()
		}
		if err := r.admin.DestroyWorkspace(ctx, "ws-del"); err != nil {
			return false, err.Error()
		}
		_, err := r.admin.GetWorkspace(ctx, "ws-del")
		return wantCode(err, codes.NotFound)
	})
}

// ---------------------------------------------------------------------------
// AgentService
// ---------------------------------------------------------------------------

func (r *run) agentChecks() {
	const g = "AgentService"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r.check(g, "agent-spawn-missing-workspace", Required, func() (bool, string) {
		_, err := r.admin.Agent.SpawnAgent(ctx, &arpv1.SpawnAgentRequest{Template: "crush"})
		return wantCode(err, codes.InvalidArgument)
	})
	r.check(g, "agent-spawn-missing-template", Required, func() (bool, string) {
		_, err := r.admin.Agent.SpawnAgent(ctx, &arpv1.SpawnAgentRequest{Workspace: "arp-test"})
		return wantCode(err, codes.InvalidArgument)
	})
	r.check(g, "agent-spawn-unknown-workspace", Required, func() (bool, string) {
		_, err := r.admin.SpawnAgent(ctx, "nonexistent-ws-00000", "crush", "x")
		return wantCode(err, codes.NotFound)
	})
	r.check(g, "agent-spawn-ok-ready", Required, func() (bool, string) {
		a, err := r.admin.SpawnAgent(ctx, "arp-test", "crush", "coder")
		if err != nil {
			return false, err.Error()
		}
		if a.GetStatus() != arpv1.AgentStatus_AGENT_STATUS_READY {
			return false, "agent not READY after spawn"
		}
		if a.GetDirectUrl() == "" {
			return false, "missing direct_url"
		}
		if a.GetTokenId() == "" || a.GetSessionId() == "" {
			return false, "missing token/session id"
		}
		return true, ""
	})
	r.check(g, "agent-spawn-direct-url-format", Required, func() (bool, string) {
		a, err := r.admin.SpawnAgent(ctx, "arp-test", "crush", "coder2")
		if err != nil {
			return false, err.Error()
		}
		return truthy(strings.HasPrefix(a.GetDirectUrl(), "http://"), "direct_url not an http URL: "+a.GetDirectUrl())
	})
	r.check(g, "agent-spawn-no-proxy-without-gateway", Required, func() (bool, string) {
		a, err := r.admin.SpawnAgent(ctx, "arp-test", "crush", "coder3")
		if err != nil {
			return false, err.Error()
		}
		return truthy(a.GetProxyUrl() == "", "proxy_url should be empty without a gateway, got: "+a.GetProxyUrl())
	})
	r.check(g, "agent-list-ok", Required, func() (bool, string) {
		_, err := r.admin.ListAgents(ctx)
		return ok(err)
	})
	r.check(g, "agent-list-contains-seeded", Required, func() (bool, string) {
		as, err := r.admin.ListAgents(ctx)
		if err != nil {
			return false, err.Error()
		}
		return truthy(hasAgent(as, "echo-agent-001") && hasAgent(as, "crush-agent-001"), "seeded agents missing")
	})
	r.check(g, "agent-list-filter-workspace", Required, func() (bool, string) {
		resp, err := r.admin.Agent.ListAgents(ctx, &arpv1.ListAgentsRequest{Workspace: "arp-test"})
		if err != nil {
			return false, err.Error()
		}
		return truthy(hasAgent(resp.GetAgents(), "echo-agent-001"), "workspace filter missing echo-agent-001")
	})
	r.check(g, "agent-list-filter-status-ready", Required, func() (bool, string) {
		resp, err := r.admin.Agent.ListAgents(ctx, &arpv1.ListAgentsRequest{Status: arpv1.AgentStatus_AGENT_STATUS_READY})
		if err != nil {
			return false, err.Error()
		}
		return truthy(hasAgent(resp.GetAgents(), "crush-agent-001"), "status filter missing crush-agent-001")
	})
	r.check(g, "agent-list-filter-template", Required, func() (bool, string) {
		resp, err := r.admin.Agent.ListAgents(ctx, &arpv1.ListAgentsRequest{Template: "echo"})
		if err != nil {
			return false, err.Error()
		}
		as := resp.GetAgents()
		return truthy(hasAgent(as, "echo-agent-001") && !hasAgent(as, "crush-agent-001"), "template filter incorrect")
	})
	r.check(g, "agent-get-ok", Required, func() (bool, string) {
		a, err := r.admin.GetAgentStatus(ctx, "crush-agent-001")
		if err != nil {
			return false, err.Error()
		}
		if a.GetTemplate() == "" || a.GetWorkspace() == "" || a.GetPort() == 0 {
			return false, "missing template/workspace/port"
		}
		if a.GetDirectUrl() == "" {
			return false, "missing direct_url"
		}
		return truthy(a.GetStatus() == arpv1.AgentStatus_AGENT_STATUS_READY, "crush-agent-001 not READY")
	})
	r.check(g, "agent-get-nonexistent", Required, func() (bool, string) {
		_, err := r.admin.GetAgentStatus(ctx, "nonexistent-00000")
		return wantCode(err, codes.NotFound)
	})
	r.check(g, "agent-stop-nonexistent", Required, func() (bool, string) {
		_, err := r.admin.StopAgent(ctx, "nonexistent-00000")
		return wantCode(err, codes.NotFound)
	})

	// Lifecycle: spawn a throwaway agent to exercise stop / restart.
	var lifecycleID string
	r.check(g, "agent-stop-ok", Required, func() (bool, string) {
		a, err := r.admin.SpawnAgent(ctx, "arp-test", "crush", "stopper")
		if err != nil {
			return false, err.Error()
		}
		lifecycleID = a.GetId()
		stopped, err := r.admin.StopAgent(ctx, lifecycleID)
		if err != nil {
			return false, err.Error()
		}
		return truthy(stopped.GetStatus() == arpv1.AgentStatus_AGENT_STATUS_STOPPED, "agent not STOPPED")
	})
	r.check(g, "agent-restart-nonexistent", Required, func() (bool, string) {
		_, err := r.admin.Agent.RestartAgent(ctx, &arpv1.RestartAgentRequest{AgentId: "nonexistent-00000"})
		return wantCode(err, codes.NotFound)
	})
	r.check(g, "agent-restart-ok", Required, func() (bool, string) {
		if lifecycleID == "" {
			return false, "no lifecycle agent"
		}
		a, err := r.admin.Agent.RestartAgent(ctx, &arpv1.RestartAgentRequest{AgentId: lifecycleID})
		if err != nil {
			return false, err.Error()
		}
		return truthy(a.GetStatus() == arpv1.AgentStatus_AGENT_STATUS_READY, "agent not READY after restart")
	})
}

// ---------------------------------------------------------------------------
// DiscoveryService
// ---------------------------------------------------------------------------

func (r *run) discoveryChecks() {
	const g = "DiscoveryService"
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	r.check(g, "discover-ok", Required, func() (bool, string) {
		resp, err := r.admin.DiscoverAgents(ctx, "")
		if err != nil {
			return false, err.Error()
		}
		return truthy(len(resp.GetAgentCards()) >= 2, "expected at least seeded agent cards")
	})
	r.check(g, "discover-local-scope", Required, func() (bool, string) {
		_, err := r.admin.Discovery.DiscoverAgents(ctx, &arpv1.DiscoverAgentsRequest{Scope: arpv1.DiscoveryScope_DISCOVERY_SCOPE_LOCAL})
		return ok(err)
	})
	r.check(g, "discover-finds-crush", Required, func() (bool, string) {
		resp, err := r.admin.DiscoverAgents(ctx, "")
		if err != nil {
			return false, err.Error()
		}
		return truthy(cardsHaveAgent(resp.GetAgentCards(), "crush-agent-001"), "crush-agent-001 not discovered")
	})
	r.check(g, "discover-capability-crush", Required, func() (bool, string) {
		resp, err := r.admin.DiscoverAgents(ctx, "crush")
		if err != nil {
			return false, err.Error()
		}
		cards := resp.GetAgentCards()
		return truthy(cardsHaveAgent(cards, "crush-agent-001") && !cardsHaveAgent(cards, "echo-agent-001"), "capability=crush filter incorrect")
	})
	r.check(g, "discover-capability-echo", Required, func() (bool, string) {
		resp, err := r.admin.DiscoverAgents(ctx, "echo")
		if err != nil {
			return false, err.Error()
		}
		return truthy(cardsHaveAgent(resp.GetAgentCards(), "echo-agent-001"), "capability=echo did not find echo-agent-001")
	})
	r.check(g, "discover-capability-nomatch", Required, func() (bool, string) {
		resp, err := r.admin.DiscoverAgents(ctx, "nonexistent-00000")
		if err != nil {
			return false, err.Error()
		}
		return truthy(len(resp.GetAgentCards()) == 0, "expected no cards for unknown capability")
	})
	r.check(g, "watch-agent-nonexistent", Required, func() (bool, string) {
		wctx, wcancel := context.WithTimeout(ctx, 5*time.Second)
		defer wcancel()
		stream, err := r.admin.Discovery.WatchAgent(wctx, &arpv1.WatchAgentRequest{AgentId: "nonexistent-00000"})
		if err == nil {
			_, err = stream.Recv()
		}
		return wantCode(err, codes.NotFound)
	})
	r.check(g, "watch-agent-initial-and-stop", Required, func() (bool, string) {
		a, err := r.admin.SpawnAgent(ctx, "arp-test", "crush", "watched")
		if err != nil {
			return false, err.Error()
		}
		wctx, wcancel := context.WithTimeout(ctx, 8*time.Second)
		defer wcancel()
		stream, err := r.admin.Discovery.WatchAgent(wctx, &arpv1.WatchAgentRequest{AgentId: a.GetId()})
		if err != nil {
			return false, err.Error()
		}
		first, err := stream.Recv()
		if err != nil {
			return false, "initial recv: " + err.Error()
		}
		if first.GetEventType() != arpv1.AgentEventType_AGENT_EVENT_TYPE_STATUS_CHANGED {
			return false, "first event not STATUS_CHANGED"
		}
		go func() { _, _ = r.admin.StopAgent(context.Background(), a.GetId()) }()
		sawStopped := false
		for {
			ev, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return false, "recv: " + err.Error()
			}
			if ev.GetEventType() == arpv1.AgentEventType_AGENT_EVENT_TYPE_STOPPED {
				sawStopped = true
				break
			}
		}
		return truthy(sawStopped, "did not observe STOPPED event")
	})
	r.check(g, "watch-workspace-initial-spawns", Recommended, func() (bool, string) {
		wctx, wcancel := context.WithTimeout(ctx, 6*time.Second)
		defer wcancel()
		stream, err := r.admin.Discovery.WatchWorkspace(wctx, &arpv1.WatchWorkspaceRequest{WorkspaceName: "arp-test"})
		if err != nil {
			return false, err.Error()
		}
		ev, err := stream.Recv()
		if err != nil {
			return false, err.Error()
		}
		return truthy(ev.GetEventType() == arpv1.WorkspaceEventType_WORKSPACE_EVENT_TYPE_AGENT_SPAWNED, "first workspace event not AGENT_SPAWNED")
	})
}

// ---------------------------------------------------------------------------
// TokenService
// ---------------------------------------------------------------------------

func (r *run) tokenChecks() {
	const g = "TokenService"
	ctx, cancel := shortCtx()
	defer cancel()

	r.check(g, "token-create-ok", Required, func() (bool, string) {
		_, err := r.admin.CreateToken(ctx, "svc", &arpv1.Scope{Global: true}, arpv1.Permission_PERMISSION_ADMIN, 0)
		return ok(err)
	})
	r.check(g, "token-create-has-bearer", Required, func() (bool, string) {
		resp, err := r.admin.CreateToken(ctx, "svc", &arpv1.Scope{Global: true}, arpv1.Permission_PERMISSION_ADMIN, 0)
		if err != nil {
			return false, err.Error()
		}
		return truthy(strings.HasPrefix(resp.GetBearerToken(), "arp_"), "bearer token missing/invalid")
	})
	r.check(g, "token-create-missing-subject", Required, func() (bool, string) {
		_, err := r.admin.Token.CreateToken(ctx, &arpv1.CreateTokenRequest{Scope: &arpv1.Scope{Global: true}, Permission: arpv1.Permission_PERMISSION_ADMIN})
		return wantCode(err, codes.InvalidArgument)
	})
	r.check(g, "token-create-missing-scope", Required, func() (bool, string) {
		_, err := r.admin.Token.CreateToken(ctx, &arpv1.CreateTokenRequest{Subject: "s", Permission: arpv1.Permission_PERMISSION_ADMIN})
		return wantCode(err, codes.InvalidArgument)
	})
	r.check(g, "token-create-missing-permission", Required, func() (bool, string) {
		_, err := r.admin.Token.CreateToken(ctx, &arpv1.CreateTokenRequest{Subject: "s", Scope: &arpv1.Scope{Global: true}})
		return wantCode(err, codes.InvalidArgument)
	})
	r.check(g, "token-create-non-admin-denied", Required, func() (bool, string) {
		tc, cl, err := r.tokenClient(ctx, "proj", scopeProjects("myapp"), arpv1.Permission_PERMISSION_PROJECT)
		if err != nil {
			return false, err.Error()
		}
		defer cl.Close()
		_, err = tc.CreateToken(ctx, "child", &arpv1.Scope{Global: true}, arpv1.Permission_PERMISSION_SESSION, 0)
		return wantCode(err, codes.PermissionDenied)
	})
	r.check(g, "token-create-scope-widen-denied", Required, func() (bool, string) {
		// An admin token narrowed to myapp cannot mint a global token.
		tc, cl, err := r.tokenClient(ctx, "scoped-admin", scopeProjects("myapp"), arpv1.Permission_PERMISSION_ADMIN)
		if err != nil {
			return false, err.Error()
		}
		defer cl.Close()
		_, err = tc.CreateToken(ctx, "child", &arpv1.Scope{Global: true}, arpv1.Permission_PERMISSION_ADMIN, 0)
		return wantCode(err, codes.PermissionDenied)
	})
}

// ---------------------------------------------------------------------------
// Identity & Scopes
// ---------------------------------------------------------------------------

func (r *run) identityChecks() {
	const g = "Identity & Scopes"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Auxiliary project + workspace out of the myapp scope.
	_, _ = r.admin.RegisterProject(ctx, "other", "/tmp/other")
	_, _ = r.admin.CreateWorkspace(ctx, "ws-other", "other")

	r.check(g, "scope-list-projects-filtered", Required, func() (bool, string) {
		tc, cl, err := r.tokenClient(ctx, "scoped", scopeProjects("myapp"), arpv1.Permission_PERMISSION_PROJECT)
		if err != nil {
			return false, err.Error()
		}
		defer cl.Close()
		ps, err := tc.ListProjects(ctx)
		if err != nil {
			return false, err.Error()
		}
		return truthy(containsProject(ps, "myapp") && !containsProject(ps, "other"), "scope filter on ListProjects incorrect")
	})
	r.check(g, "scope-workspace-out-of-scope-denied", Required, func() (bool, string) {
		tc, cl, err := r.tokenClient(ctx, "scoped", scopeProjects("myapp"), arpv1.Permission_PERMISSION_PROJECT)
		if err != nil {
			return false, err.Error()
		}
		defer cl.Close()
		_, err = tc.GetWorkspace(ctx, "ws-other")
		return wantCode(err, codes.PermissionDenied)
	})
	r.check(g, "spawn-scope-widen-denied", Required, func() (bool, string) {
		tc, cl, err := r.tokenClient(ctx, "scoped", scopeProjects("myapp"), arpv1.Permission_PERMISSION_PROJECT)
		if err != nil {
			return false, err.Error()
		}
		defer cl.Close()
		_, err = tc.Agent.SpawnAgent(ctx, &arpv1.SpawnAgentRequest{Workspace: "arp-test", Template: "crush", Name: "widen", Scope: &arpv1.Scope{Global: true}})
		return wantCode(err, codes.PermissionDenied)
	})
	r.check(g, "spawn-perm-elevate-denied", Required, func() (bool, string) {
		tc, cl, err := r.tokenClient(ctx, "scoped", scopeProjects("myapp"), arpv1.Permission_PERMISSION_PROJECT)
		if err != nil {
			return false, err.Error()
		}
		defer cl.Close()
		_, err = tc.Agent.SpawnAgent(ctx, &arpv1.SpawnAgentRequest{Workspace: "arp-test", Template: "crush", Name: "elevate", Permission: arpv1.Permission_PERMISSION_ADMIN})
		return wantCode(err, codes.PermissionDenied)
	})
	r.check(g, "spawn-scope-narrow-ok", Required, func() (bool, string) {
		tc, cl, err := r.tokenClient(ctx, "scoped", scopeProjects("myapp"), arpv1.Permission_PERMISSION_PROJECT)
		if err != nil {
			return false, err.Error()
		}
		defer cl.Close()
		a, err := tc.Agent.SpawnAgent(ctx, &arpv1.SpawnAgentRequest{Workspace: "arp-test", Template: "crush", Name: "narrowed", Scope: scopeProjects("myapp")})
		if err != nil {
			return false, err.Error()
		}
		return truthy(a.GetStatus() == arpv1.AgentStatus_AGENT_STATUS_READY, "narrowed spawn not ready")
	})

	// Session isolation between two SESSION tokens.
	r.check(g, "session-isolation", Required, func() (bool, string) {
		taA, clA, err := r.tokenClient(ctx, "sessA", scopeProjects("myapp"), arpv1.Permission_PERMISSION_SESSION)
		if err != nil {
			return false, err.Error()
		}
		defer clA.Close()
		taB, clB, err := r.tokenClient(ctx, "sessB", scopeProjects("myapp"), arpv1.Permission_PERMISSION_SESSION)
		if err != nil {
			return false, err.Error()
		}
		defer clB.Close()

		a1, err := taA.SpawnAgent(ctx, "arp-test", "crush", "a1")
		if err != nil {
			return false, "A spawn: " + err.Error()
		}
		listA, err := taA.ListAgents(ctx)
		if err != nil {
			return false, err.Error()
		}
		if !hasAgent(listA, a1.GetId()) {
			return false, "A cannot see its own agent"
		}
		listB, err := taB.ListAgents(ctx)
		if err != nil {
			return false, err.Error()
		}
		if hasAgent(listB, a1.GetId()) {
			return false, "B can see A's agent (isolation broken)"
		}
		_, err = taB.StopAgent(ctx, a1.GetId())
		if status.Code(err) != codes.PermissionDenied {
			return false, fmt.Sprintf("B stop A's agent expected PermissionDenied, got %v", err)
		}
		if _, err := taA.GetAgentStatus(ctx, a1.GetId()); err != nil {
			return false, "A get own agent: " + err.Error()
		}
		return true, ""
	})
	r.check(g, "session-propagation", Required, func() (bool, string) {
		ta, cl, err := r.tokenClient(ctx, "sessP", scopeProjects("myapp"), arpv1.Permission_PERMISSION_SESSION)
		if err != nil {
			return false, err.Error()
		}
		defer cl.Close()
		a1, err := ta.SpawnAgent(ctx, "arp-test", "crush", "p1")
		if err != nil {
			return false, err.Error()
		}
		a2, err := ta.SpawnAgent(ctx, "arp-test", "crush", "p2")
		if err != nil {
			return false, err.Error()
		}
		if a1.GetSessionId() == "" {
			return false, "missing session id"
		}
		return truthy(a1.GetSessionId() == a2.GetSessionId(), "sibling agents have different session ids")
	})
	r.check(g, "project-perm-sees-all-in-project", Recommended, func() (bool, string) {
		tc, cl, err := r.tokenClient(ctx, "lead", scopeProjects("myapp"), arpv1.Permission_PERMISSION_PROJECT)
		if err != nil {
			return false, err.Error()
		}
		defer cl.Close()
		as, err := tc.ListAgents(ctx)
		if err != nil {
			return false, err.Error()
		}
		// Seeded agents were spawned by the admin (different session); a PROJECT
		// token must still see them.
		return truthy(hasAgent(as, "crush-agent-001"), "PROJECT token cannot see project agents")
	})
}

// ---------------------------------------------------------------------------
// Two-interface & AgentCard enrichment
// ---------------------------------------------------------------------------

func (r *run) enrichmentChecks() {
	const g = "Two-Interface & Enrichment"
	ctx, cancel := shortCtx()
	defer cancel()

	r.check(g, "enrich-no-proxy-without-gateway", Required, func() (bool, string) {
		a, err := r.admin.GetAgentStatus(ctx, "crush-agent-001")
		if err != nil {
			return false, err.Error()
		}
		return truthy(a.GetProxyUrl() == "", "proxy_url should be empty without a gateway, got: "+a.GetProxyUrl())
	})
	r.check(g, "enrich-direct-url", Required, func() (bool, string) {
		a, err := r.admin.GetAgentStatus(ctx, "crush-agent-001")
		if err != nil {
			return false, err.Error()
		}
		return truthy(strings.HasPrefix(a.GetDirectUrl(), "http://127.0.0.1:"), "unexpected direct_url: "+a.GetDirectUrl())
	})
	r.check(g, "enrich-card-arp-metadata", Required, func() (bool, string) {
		a, err := r.admin.GetAgentStatus(ctx, "crush-agent-001")
		if err != nil {
			return false, err.Error()
		}
		card := a.GetA2AAgentCard()
		if card == nil {
			return false, "missing a2a_agent_card"
		}
		arp, ok := card.GetMetadata().AsMap()["arp"].(map[string]any)
		if !ok {
			return false, "missing metadata.arp"
		}
		if arp["agent_id"] != "crush-agent-001" {
			return false, "metadata.arp.agent_id mismatch"
		}
		if arp["direct_url"] != a.GetDirectUrl() {
			return false, "metadata.arp.direct_url mismatch"
		}
		if arp["status"] != "ready" {
			return false, fmt.Sprintf("metadata.arp.status = %v, want ready", arp["status"])
		}
		return truthy(arp["workspace"] == "arp-test" && arp["project"] == "myapp", "metadata.arp workspace/project mismatch")
	})
	r.check(g, "enrich-card-supported-interface-direct", Required, func() (bool, string) {
		a, err := r.admin.GetAgentStatus(ctx, "crush-agent-001")
		if err != nil {
			return false, err.Error()
		}
		card := a.GetA2AAgentCard()
		if card == nil {
			return false, "missing a2a_agent_card"
		}
		ifaces := card.GetSupportedInterfaces()
		if len(ifaces) == 0 {
			return false, "missing supported_interfaces"
		}
		// Direct is the canonical interface: the AgentCard points clients at the
		// agent's own A2A endpoint, not at ARP.
		return truthy(ifaces[0].GetUrl() == a.GetDirectUrl(), "supported_interfaces[0].url != direct_url")
	})
}

// ---------------------------------------------------------------------------
// Optional A2A gateway profile (proxy_url enrichment when a gateway is present)
// ---------------------------------------------------------------------------

func (r *run) gatewayChecks() {
	const g = "Gateway Profile (optional)"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const gatewayURL = "http://gateway.example:9099"
	env, err := StartWithGateway(ctx, gatewayURL)
	if err != nil {
		r.check(g, "gateway-start", Recommended, func() (bool, string) { return false, err.Error() })
		return
	}
	defer env.Stop()
	cl, err := env.Admin()
	if err != nil {
		r.check(g, "gateway-connect", Recommended, func() (bool, string) { return false, err.Error() })
		return
	}
	defer cl.Close()

	r.check(g, "gateway-proxy-url-populated", Recommended, func() (bool, string) {
		a, err := cl.GetAgentStatus(ctx, "crush-agent-001")
		if err != nil {
			return false, err.Error()
		}
		want := gatewayURL + "/a2a/agents/crush-agent-001"
		return truthy(a.GetProxyUrl() == want, "proxy_url = "+a.GetProxyUrl()+", want "+want)
	})
	r.check(g, "gateway-card-direct-still-primary", Recommended, func() (bool, string) {
		a, err := cl.GetAgentStatus(ctx, "crush-agent-001")
		if err != nil {
			return false, err.Error()
		}
		ifaces := a.GetA2AAgentCard().GetSupportedInterfaces()
		if len(ifaces) < 2 {
			return false, "expected direct + gateway interfaces"
		}
		return truthy(ifaces[0].GetUrl() == a.GetDirectUrl() && ifaces[1].GetUrl() == a.GetProxyUrl(),
			"interface order should be [direct, gateway]")
	})
	r.check(g, "gateway-card-metadata-proxy-url", Recommended, func() (bool, string) {
		a, err := cl.GetAgentStatus(ctx, "crush-agent-001")
		if err != nil {
			return false, err.Error()
		}
		arp, ok := a.GetA2AAgentCard().GetMetadata().AsMap()["arp"].(map[string]any)
		if !ok {
			return false, "missing metadata.arp"
		}
		return truthy(arp["proxy_url"] == a.GetProxyUrl(), "metadata.arp.proxy_url mismatch")
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (r *run) tokenClient(ctx context.Context, subject string, scope *arpv1.Scope, perm arpv1.Permission) (*client.Client, *client.Client, error) {
	resp, err := r.admin.CreateToken(ctx, subject, scope, perm, 0)
	if err != nil {
		return nil, nil, err
	}
	cl, err := r.env.TokenClient(resp.GetBearerToken())
	if err != nil {
		return nil, nil, err
	}
	return cl, cl, nil
}

func ok(err error) (bool, string) {
	if err != nil {
		return false, err.Error()
	}
	return true, ""
}

func wantCode(err error, want codes.Code) (bool, string) {
	got := status.Code(err)
	if got == want {
		return true, ""
	}
	return false, fmt.Sprintf("expected %s, got %s (%v)", want, got, err)
}

func truthy(cond bool, detail string) (bool, string) {
	if cond {
		return true, ""
	}
	return false, detail
}

func scopeProjects(projects ...string) *arpv1.Scope {
	return &arpv1.Scope{Projects: projects}
}

func containsProject(ps []*arpv1.Project, name string) bool {
	for _, p := range ps {
		if p.GetName() == name {
			return true
		}
	}
	return false
}

func containsWorkspace(ws []*arpv1.Workspace, name string) bool {
	for _, w := range ws {
		if w.GetName() == name {
			return true
		}
	}
	return false
}

func hasAgent(as []*arpv1.AgentInstance, id string) bool {
	for _, a := range as {
		if a.GetId() == id {
			return true
		}
	}
	return false
}

func cardsHaveAgent(cards []*a2av1.AgentCard, id string) bool {
	for _, c := range cards {
		if arp, ok := c.GetMetadata().AsMap()["arp"].(map[string]any); ok && arp["agent_id"] == id {
			return true
		}
	}
	return false
}
