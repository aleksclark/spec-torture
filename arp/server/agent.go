package server

import (
	"context"
	"sort"

	"github.com/aleksclark/spec-torture/arp/backend"
	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// proxyURL returns the optional gateway URL for an agent id, or "" when no A2A
// gateway is configured. ARP core does not proxy A2A traffic; the gateway is an
// optional deployment component (see arp-spec/profile-a2a-gateway.md).
func (s *Server) proxyURL(agentID string) string {
	if s.cfg.GatewayBaseURL == "" {
		return ""
	}
	return s.cfg.GatewayBaseURL + "/a2a/agents/" + agentID
}

// resolveTemplate finds a template by name in a project, then in the global
// template set. Caller must hold s.mu.
func (s *Server) resolveTemplate(project, name string) *arpv1.AgentTemplate {
	if proj := s.projects[project]; proj != nil {
		for _, t := range proj.GetAgents() {
			if t.GetName() == name {
				return t
			}
		}
	}
	if t := s.cfg.Templates[name]; t != nil {
		return t
	}
	return nil
}

// resolveSession returns the session id for a spawn, establishing one on the
// caller's token on first use. Caller must hold s.mu.
func (s *Server) resolveSession(p *principal) string {
	if p.token != nil {
		if p.token.sessionID == "" {
			p.token.sessionID = newID("sess")
		}
		p.sessionID = p.token.sessionID
		return p.token.sessionID
	}
	sess := newID("sess")
	if p.sessionID == "" {
		p.sessionID = sess
	}
	return sess
}

// findAgentForCaller resolves an agent and enforces visibility. Caller must
// hold s.mu.
func (s *Server) findAgentForCaller(p *principal, id string) (*agentEntry, error) {
	a, ok := s.agents[id]
	if !ok {
		return nil, errNotFound("agent %q not found", id)
	}
	if !s.canSeeAgent(p, a) {
		return nil, errPermissionDenied("agent %q not accessible", id)
	}
	return a, nil
}

// spawnAgentInternal performs the full spawn lifecycle: validate scope, allocate
// a port, issue a child token, start the process via the backend, wait for
// readiness, then resolve and enrich the agent's AgentCard.
func (s *Server) spawnAgentInternal(ctx context.Context, p *principal, workspace, template, name string, env map[string]string, reqScope *arpv1.Scope, reqPerm arpv1.Permission, fixedID string) (*arpv1.AgentInstance, error) {
	s.mu.Lock()
	ws, ok := s.workspaces[workspace]
	if !ok {
		s.mu.Unlock()
		return nil, errNotFound("workspace %q not found", workspace)
	}
	project := ws.project
	if !p.canAccessProject(project) {
		s.mu.Unlock()
		return nil, errPermissionDenied("project %q not in token scope", project)
	}
	tmpl := s.resolveTemplate(project, template)
	if tmpl == nil {
		s.mu.Unlock()
		return nil, errNotFound("template %q not found in project %q", template, project)
	}
	port, err := s.allocPort()
	if err != nil {
		s.mu.Unlock()
		return nil, errInternal("allocate port: %v", err)
	}
	session := s.resolveSession(p)
	childTok, err := s.issueToken(p, reqScope, reqPerm, displayName(name, template), session, 0)
	if err != nil {
		s.freePort(port)
		s.mu.Unlock()
		return nil, err
	}
	id := fixedID
	if id == "" {
		id = newID(displayName(name, template))
	}
	if _, exists := s.agents[id]; exists {
		s.freePort(port)
		s.mu.Unlock()
		return nil, errAlreadyExists("agent %q already exists", id)
	}
	spawnedBy := ""
	if p.token != nil {
		spawnedBy = p.token.id
	}
	entry := &agentEntry{
		pb: &arpv1.AgentInstance{
			Id:        id,
			Template:  template,
			Workspace: workspace,
			Status:    arpv1.AgentStatus_AGENT_STATUS_STARTING,
			Port:      int32(port),
			ProxyUrl:  s.proxyURL(id),
			TokenId:   childTok.id,
			SessionId: session,
			SpawnedBy: spawnedBy,
			StartedAt: nowTS(),
		},
		template: tmpl,
		project:  project,
		name:     displayName(name, template),
		env:      env,
	}
	s.agents[id] = entry
	ws.agents[id] = true
	s.mu.Unlock()

	handle, spawnErr := s.cfg.Backend.Spawn(ctx, backend.SpawnSpec{
		AgentID:      id,
		Template:     tmpl,
		Workspace:    workspace,
		WorkspaceDir: ws.dir,
		Port:         port,
		Env:          env,
		Token:        childTok.bearer,
	})
	if spawnErr != nil {
		s.mu.Lock()
		a := s.agents[id]
		if a != nil {
			a.pb.Status = arpv1.AgentStatus_AGENT_STATUS_ERROR
		}
		s.freePort(port)
		s.mu.Unlock()
		return nil, errInternal("spawn agent %q: %v", id, spawnErr)
	}

	// Fetch the agent's own AgentCard (no lock held) to seed the enriched card.
	// This is the only A2A call ARP makes — discovery, never messaging.
	card, cardErr := s.a2a.FetchAgentCard(ctx, handle.DirectURL())
	if cardErr != nil || card == nil {
		card = backendCardFallback(tmpl, handle.DirectURL())
	}

	s.mu.Lock()
	entry.handle = handle
	entry.cardBase = card
	entry.pb.DirectUrl = handle.DirectURL()
	entry.pb.Port = int32(handle.Port())
	entry.pb.Pid = int32(handle.PID())
	entry.pb.Status = arpv1.AgentStatus_AGENT_STATUS_READY
	s.refreshCard(entry)
	result := cloneAgent(entry.pb)
	s.mu.Unlock()

	s.publishWorkspaceSpawn(ws, entry)
	s.publishAgentEvent(entry, arpv1.AgentEventType_AGENT_EVENT_TYPE_STATUS_CHANGED, nil)
	return result, nil
}

// SpawnAgent spawns a new agent in a workspace.
func (s *Server) SpawnAgent(ctx context.Context, req *arpv1.SpawnAgentRequest) (*arpv1.AgentInstance, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetWorkspace() == "" {
		return nil, errInvalidArgument("workspace is required")
	}
	if req.GetTemplate() == "" {
		return nil, errInvalidArgument("template is required")
	}
	return s.spawnAgentInternal(ctx, p, req.GetWorkspace(), req.GetTemplate(), req.GetName(), req.GetEnv(), req.GetScope(), req.GetPermission(), "")
}

// ListAgents lists agents visible to the caller, with optional filters.
func (s *Server) ListAgents(ctx context.Context, req *arpv1.ListAgentsRequest) (*arpv1.ListAgentsResponse, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := sortedAgentIDs(s.agents)
	resp := &arpv1.ListAgentsResponse{}
	for _, id := range ids {
		a := s.agents[id]
		if !s.canSeeAgent(p, a) {
			continue
		}
		if req.GetWorkspace() != "" && a.pb.GetWorkspace() != req.GetWorkspace() {
			continue
		}
		if req.GetStatus() != arpv1.AgentStatus_AGENT_STATUS_UNSPECIFIED && a.pb.GetStatus() != req.GetStatus() {
			continue
		}
		if req.GetTemplate() != "" && a.pb.GetTemplate() != req.GetTemplate() {
			continue
		}
		resp.Agents = append(resp.Agents, cloneAgent(a.pb))
	}
	return resp, nil
}

// GetAgentStatus returns the current status of a single agent.
func (s *Server) GetAgentStatus(ctx context.Context, req *arpv1.GetAgentStatusRequest) (*arpv1.AgentInstance, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetAgentId() == "" {
		return nil, errInvalidArgument("agent_id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.findAgentForCaller(p, req.GetAgentId())
	if err != nil {
		return nil, err
	}
	return cloneAgent(a.pb), nil
}

// StopAgent gracefully stops an agent.
func (s *Server) StopAgent(ctx context.Context, req *arpv1.StopAgentRequest) (*arpv1.AgentInstance, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetAgentId() == "" {
		return nil, errInvalidArgument("agent_id is required")
	}
	s.mu.Lock()
	a, err := s.findAgentForCaller(p, req.GetAgentId())
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Unlock()

	grace := int(req.GetGracePeriodMs())
	if grace <= 0 {
		grace = 5000
	}
	if err := s.stopAgentInternal(ctx, req.GetAgentId(), grace); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneAgent(a.pb), nil
}

// stopAgentInternal stops an agent process and updates its status.
func (s *Server) stopAgentInternal(ctx context.Context, id string, graceMs int) error {
	s.mu.Lock()
	a, ok := s.agents[id]
	if !ok {
		s.mu.Unlock()
		return nil
	}
	handle := a.handle
	a.pb.Status = arpv1.AgentStatus_AGENT_STATUS_STOPPING
	s.refreshCard(a)
	s.mu.Unlock()
	s.publishAgentEvent(a, arpv1.AgentEventType_AGENT_EVENT_TYPE_STATUS_CHANGED, nil)

	if handle != nil {
		_ = handle.Stop(ctx, graceMs)
	}

	s.mu.Lock()
	a.pb.Status = arpv1.AgentStatus_AGENT_STATUS_STOPPED
	a.pb.Pid = 0
	s.freePort(int(a.pb.GetPort()))
	a.handle = nil
	s.refreshCard(a)
	s.mu.Unlock()
	s.publishAgentEvent(a, arpv1.AgentEventType_AGENT_EVENT_TYPE_STOPPED, nil)
	return nil
}

// RestartAgent stops and re-spawns an agent with the same configuration.
// A new port may be allocated, so the agent's direct_url may change.
func (s *Server) RestartAgent(ctx context.Context, req *arpv1.RestartAgentRequest) (*arpv1.AgentInstance, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetAgentId() == "" {
		return nil, errInvalidArgument("agent_id is required")
	}
	s.mu.Lock()
	a, err := s.findAgentForCaller(p, req.GetAgentId())
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	id := a.pb.GetId()
	workspace := a.pb.GetWorkspace()
	env := a.env
	ws := s.workspaces[workspace]
	s.mu.Unlock()

	_ = s.stopAgentInternal(ctx, id, 5000)

	s.mu.Lock()
	port, perr := s.allocPort()
	if perr != nil {
		s.mu.Unlock()
		return nil, errInternal("allocate port: %v", perr)
	}
	s.mu.Unlock()

	handle, spawnErr := s.cfg.Backend.Spawn(ctx, backend.SpawnSpec{
		AgentID:      id,
		Template:     a.template,
		Workspace:    workspace,
		WorkspaceDir: dirOf(ws),
		Port:         port,
		Env:          env,
	})
	if spawnErr != nil {
		s.mu.Lock()
		a.pb.Status = arpv1.AgentStatus_AGENT_STATUS_ERROR
		s.freePort(port)
		s.mu.Unlock()
		return nil, errInternal("restart agent %q: %v", id, spawnErr)
	}
	card, cardErr := s.a2a.FetchAgentCard(ctx, handle.DirectURL())
	if cardErr != nil || card == nil {
		card = backendCardFallback(a.template, handle.DirectURL())
	}

	s.mu.Lock()
	a.handle = handle
	a.cardBase = card
	a.pb.DirectUrl = handle.DirectURL()
	a.pb.Port = int32(handle.Port())
	a.pb.Pid = int32(handle.PID())
	a.pb.Status = arpv1.AgentStatus_AGENT_STATUS_READY
	a.pb.StartedAt = nowTS()
	s.refreshCard(a)
	result := cloneAgent(a.pb)
	enriched := a.pb.GetA2AAgentCard()
	s.mu.Unlock()

	s.publishAgentEvent(a, arpv1.AgentEventType_AGENT_EVENT_TYPE_CARD_UPDATED, enriched)
	s.publishAgentEvent(a, arpv1.AgentEventType_AGENT_EVENT_TYPE_STATUS_CHANGED, nil)
	return result, nil
}

// ---------------------------------------------------------------------------
// small helpers
// ---------------------------------------------------------------------------

func displayName(name, template string) string {
	if name != "" {
		return name
	}
	return template
}

func dirOf(ws *workspaceEntry) string {
	if ws == nil {
		return ""
	}
	return ws.dir
}

func sortedAgentIDs(m map[string]*agentEntry) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func backendCardFallback(t *arpv1.AgentTemplate, directURL string) map[string]any {
	tags := make([]any, 0)
	for _, c := range backend.SkillTags(t) {
		tags = append(tags, c)
	}
	name := t.GetName()
	desc := ""
	if cfg := t.GetA2ACardConfig(); cfg != nil {
		if cfg.GetName() != "" {
			name = cfg.GetName()
		}
		desc = cfg.GetDescription()
	}
	return map[string]any{
		"name":        name,
		"description": desc,
		"version":     "1.0.0",
		"supported_interfaces": []any{
			map[string]any{"url": directURL, "transport": "HTTP_JSON"},
		},
		"skills": []any{map[string]any{
			"id":   t.GetName(),
			"name": name,
			"tags": tags,
		}},
		"default_input_modes":  []any{"text/plain"},
		"default_output_modes": []any{"text/plain"},
	}
}
