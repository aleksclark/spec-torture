package server

import (
	"context"
	"path/filepath"
	"sort"

	"google.golang.org/protobuf/types/known/emptypb"

	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// CreateWorkspace creates a new workspace for a registered project and
// optionally auto-spawns agents.
func (s *Server) CreateWorkspace(ctx context.Context, req *arpv1.CreateWorkspaceRequest) (*arpv1.Workspace, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetName() == "" {
		return nil, errInvalidArgument("name is required")
	}
	if req.GetProject() == "" {
		return nil, errInvalidArgument("project is required")
	}

	s.mu.Lock()
	proj, ok := s.projects[req.GetProject()]
	if !ok {
		s.mu.Unlock()
		return nil, errNotFound("project %q not found", req.GetProject())
	}
	if !p.canAccessProject(req.GetProject()) {
		s.mu.Unlock()
		return nil, errPermissionDenied("project %q not in token scope", req.GetProject())
	}
	if _, exists := s.workspaces[req.GetName()]; exists {
		s.mu.Unlock()
		return nil, errAlreadyExists("workspace %q already exists", req.GetName())
	}
	branch := req.GetBranch()
	if branch == "" {
		branch = proj.GetBranch()
	}
	ws := &workspaceEntry{
		pb: &arpv1.Workspace{
			Name:      req.GetName(),
			Project:   req.GetProject(),
			Dir:       filepath.Join(s.cfg.WorkspaceRoot, req.GetName()),
			Status:    arpv1.WorkspaceStatus_WORKSPACE_STATUS_ACTIVE,
			CreatedAt: nowTS(),
		},
		project: req.GetProject(),
		dir:     filepath.Join(s.cfg.WorkspaceRoot, req.GetName()),
		agents:  map[string]bool{},
	}
	s.workspaces[req.GetName()] = ws
	s.mu.Unlock()

	for _, tmplName := range req.GetAutoAgents() {
		if _, err := s.spawnAgentInternal(ctx, p, req.GetName(), tmplName, "", nil, nil, arpv1.Permission_PERMISSION_UNSPECIFIED, ""); err != nil {
			return nil, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buildWorkspacePB(ws), nil
}

// ListWorkspaces lists workspaces visible to the caller, with optional project
// and status filters.
func (s *Server) ListWorkspaces(ctx context.Context, req *arpv1.ListWorkspacesRequest) (*arpv1.ListWorkspacesResponse, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0, len(s.workspaces))
	for name := range s.workspaces {
		names = append(names, name)
	}
	sort.Strings(names)
	resp := &arpv1.ListWorkspacesResponse{}
	for _, name := range names {
		ws := s.workspaces[name]
		if !s.workspaceVisible(p, ws) {
			continue
		}
		if req.GetProject() != "" && ws.project != req.GetProject() {
			continue
		}
		if req.GetStatus() != arpv1.WorkspaceStatus_WORKSPACE_STATUS_UNSPECIFIED && ws.pb.GetStatus() != req.GetStatus() {
			continue
		}
		resp.Workspaces = append(resp.Workspaces, s.buildWorkspacePB(ws))
	}
	return resp, nil
}

// GetWorkspace returns a single workspace by name.
func (s *Server) GetWorkspace(ctx context.Context, req *arpv1.GetWorkspaceRequest) (*arpv1.Workspace, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetName() == "" {
		return nil, errInvalidArgument("name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ws, ok := s.workspaces[req.GetName()]
	if !ok {
		return nil, errNotFound("workspace %q not found", req.GetName())
	}
	if !s.workspaceVisible(p, ws) {
		return nil, errPermissionDenied("workspace %q not accessible", req.GetName())
	}
	return s.buildWorkspacePB(ws), nil
}

// DestroyWorkspace stops all agents in a workspace and removes it.
func (s *Server) DestroyWorkspace(ctx context.Context, req *arpv1.DestroyWorkspaceRequest) (*emptypb.Empty, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetName() == "" {
		return nil, errInvalidArgument("name is required")
	}
	s.mu.Lock()
	ws, ok := s.workspaces[req.GetName()]
	if !ok {
		s.mu.Unlock()
		return nil, errNotFound("workspace %q not found", req.GetName())
	}
	if !s.workspaceVisible(p, ws) {
		s.mu.Unlock()
		return nil, errPermissionDenied("workspace %q not accessible", req.GetName())
	}
	agentIDs := make([]string, 0, len(ws.agents))
	for id := range ws.agents {
		agentIDs = append(agentIDs, id)
	}
	s.mu.Unlock()

	for _, id := range agentIDs {
		_ = s.stopAgentInternal(ctx, id, 5000)
	}

	s.mu.Lock()
	for _, id := range agentIDs {
		delete(s.agents, id)
	}
	ws.pb.Status = arpv1.WorkspaceStatus_WORKSPACE_STATUS_INACTIVE
	delete(s.workspaces, req.GetName())
	s.mu.Unlock()

	s.publishWorkspaceDestroyed(ws)
	return &emptypb.Empty{}, nil
}
