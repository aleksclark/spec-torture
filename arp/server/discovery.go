package server

import (
	"context"

	"google.golang.org/grpc"

	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// DiscoverAgents returns enriched AgentCards for agents matching the request.
func (s *Server) DiscoverAgents(ctx context.Context, req *arpv1.DiscoverAgentsRequest) (*arpv1.DiscoverAgentsResponse, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	resp := &arpv1.DiscoverAgentsResponse{}

	s.mu.Lock()
	for _, id := range sortedAgentIDs(s.agents) {
		a := s.agents[id]
		if !s.canSeeAgent(p, a) {
			continue
		}
		st := a.pb.GetStatus()
		if st != arpv1.AgentStatus_AGENT_STATUS_READY && st != arpv1.AgentStatus_AGENT_STATUS_BUSY {
			continue
		}
		if cap := req.GetCapability(); cap != "" && !cardHasCapability(a.cardBase, cap) {
			continue
		}
		if card := s.enrichedCard(a); card != nil {
			resp.AgentCards = append(resp.AgentCards, card)
		}
	}
	s.mu.Unlock()

	if req.GetScope() == arpv1.DiscoveryScope_DISCOVERY_SCOPE_NETWORK {
		for _, url := range req.GetUrls() {
			card, err := s.a2a.FetchAgentCard(ctx, url)
			if err != nil || card == nil {
				continue
			}
			if cap := req.GetCapability(); cap != "" && !cardHasCapability(card, cap) {
				continue
			}
			if cardPB, convErr := mapToAgentCard(card); convErr == nil && cardPB != nil {
				resp.AgentCards = append(resp.AgentCards, cardPB)
			}
		}
	}
	return resp, nil
}

// WatchAgent streams real-time events for a single agent.
func (s *Server) WatchAgent(req *arpv1.WatchAgentRequest, stream grpc.ServerStreamingServer[arpv1.AgentEvent]) error {
	ctx := stream.Context()
	p, err := s.authenticate(ctx)
	if err != nil {
		return err
	}
	if req.GetAgentId() == "" {
		return errInvalidArgument("agent_id is required")
	}
	s.mu.Lock()
	a, err := s.findAgentForCaller(p, req.GetAgentId())
	if err != nil {
		s.mu.Unlock()
		return err
	}
	initial := &arpv1.AgentEvent{
		EventType: arpv1.AgentEventType_AGENT_EVENT_TYPE_STATUS_CHANGED,
		Agent:     cloneAgent(a.pb),
		AgentCard: a.pb.GetA2AAgentCard(),
	}
	s.mu.Unlock()

	subID, ch := s.subscribeAgent(req.GetAgentId())
	defer s.unsubscribeAgent(req.GetAgentId(), subID)

	if err := stream.Send(initial); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-ch:
			if err := stream.Send(ev); err != nil {
				return err
			}
			if ev.GetEventType() == arpv1.AgentEventType_AGENT_EVENT_TYPE_STOPPED {
				return nil
			}
		}
	}
}

// WatchWorkspace streams real-time events for a workspace.
func (s *Server) WatchWorkspace(req *arpv1.WatchWorkspaceRequest, stream grpc.ServerStreamingServer[arpv1.WorkspaceEvent]) error {
	ctx := stream.Context()
	p, err := s.authenticate(ctx)
	if err != nil {
		return err
	}
	if req.GetWorkspaceName() == "" {
		return errInvalidArgument("workspace_name is required")
	}
	s.mu.Lock()
	ws, ok := s.workspaces[req.GetWorkspaceName()]
	if !ok {
		s.mu.Unlock()
		return errNotFound("workspace %q not found", req.GetWorkspaceName())
	}
	if !s.workspaceVisible(p, ws) {
		s.mu.Unlock()
		return errPermissionDenied("workspace %q not accessible", req.GetWorkspaceName())
	}
	var initials []*arpv1.WorkspaceEvent
	wsPB := s.buildWorkspacePB(ws)
	for _, id := range sortedAgentIDs(s.agents) {
		a := s.agents[id]
		if a.pb.GetWorkspace() != req.GetWorkspaceName() {
			continue
		}
		initials = append(initials, &arpv1.WorkspaceEvent{
			EventType: arpv1.WorkspaceEventType_WORKSPACE_EVENT_TYPE_AGENT_SPAWNED,
			Workspace: wsPB,
			Agent:     cloneAgent(a.pb),
		})
	}
	s.mu.Unlock()

	subID, ch := s.subscribeWorkspace(req.GetWorkspaceName())
	defer s.unsubscribeWorkspace(req.GetWorkspaceName(), subID)

	for _, ev := range initials {
		if err := stream.Send(ev); err != nil {
			return err
		}
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-ch:
			if err := stream.Send(ev); err != nil {
				return err
			}
			if ev.GetEventType() == arpv1.WorkspaceEventType_WORKSPACE_EVENT_TYPE_WORKSPACE_DESTROYED {
				return nil
			}
		}
	}
}

// cardHasCapability reports whether a base AgentCard advertises a skill tag
// matching the given capability.
func cardHasCapability(card map[string]any, capability string) bool {
	if card == nil {
		return false
	}
	skills, ok := card["skills"].([]any)
	if !ok {
		return false
	}
	for _, sk := range skills {
		sm, ok := sk.(map[string]any)
		if !ok {
			continue
		}
		tags, ok := sm["tags"].([]any)
		if !ok {
			continue
		}
		for _, t := range tags {
			if ts, ok := t.(string); ok && ts == capability {
				return true
			}
		}
	}
	return false
}
