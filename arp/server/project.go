package server

import (
	"context"
	"sort"

	"google.golang.org/protobuf/types/known/emptypb"

	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// ListProjects returns all projects visible within the caller's token scope.
func (s *Server) ListProjects(ctx context.Context, _ *arpv1.ListProjectsRequest) (*arpv1.ListProjectsResponse, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0, len(s.projects))
	for name := range s.projects {
		names = append(names, name)
	}
	sort.Strings(names)
	resp := &arpv1.ListProjectsResponse{}
	for _, name := range names {
		if !p.canAccessProject(name) {
			continue
		}
		resp.Projects = append(resp.Projects, cloneProject(s.projects[name]))
	}
	return resp, nil
}

// RegisterProject registers (or updates) a project. Requires admin permission.
func (s *Server) RegisterProject(ctx context.Context, req *arpv1.RegisterProjectRequest) (*arpv1.Project, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetName() == "" {
		return nil, errInvalidArgument("name is required")
	}
	if req.GetRepo() == "" {
		return nil, errInvalidArgument("repo is required")
	}
	if p.perm != arpv1.Permission_PERMISSION_ADMIN {
		return nil, errPermissionDenied("RegisterProject requires admin permission")
	}
	if !p.canAccessProject(req.GetName()) {
		return nil, errPermissionDenied("project %q not in token scope", req.GetName())
	}
	branch := req.GetBranch()
	if branch == "" {
		branch = "main"
	}
	proj := &arpv1.Project{
		Name:   req.GetName(),
		Repo:   req.GetRepo(),
		Branch: branch,
		Agents: req.GetAgents(),
	}
	s.mu.Lock()
	s.projects[req.GetName()] = proj
	s.mu.Unlock()
	return cloneProject(proj), nil
}

// UnregisterProject removes a project. Requires admin; fails if agents are
// still running for the project.
func (s *Server) UnregisterProject(ctx context.Context, req *arpv1.UnregisterProjectRequest) (*emptypb.Empty, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetName() == "" {
		return nil, errInvalidArgument("name is required")
	}
	if p.perm != arpv1.Permission_PERMISSION_ADMIN {
		return nil, errPermissionDenied("UnregisterProject requires admin permission")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[req.GetName()]; !ok {
		return nil, errNotFound("project %q not found", req.GetName())
	}
	if !p.canAccessProject(req.GetName()) {
		return nil, errPermissionDenied("project %q not in token scope", req.GetName())
	}
	for _, a := range s.agents {
		if a.project == req.GetName() && isActive(a.pb.GetStatus()) {
			return nil, errFailedPrecondition("project %q has active agents; stop them first", req.GetName())
		}
	}
	delete(s.projects, req.GetName())
	return &emptypb.Empty{}, nil
}

func isActive(st arpv1.AgentStatus) bool {
	switch st {
	case arpv1.AgentStatus_AGENT_STATUS_STOPPED, arpv1.AgentStatus_AGENT_STATUS_ERROR,
		arpv1.AgentStatus_AGENT_STATUS_UNSPECIFIED:
		return false
	default:
		return true
	}
}
