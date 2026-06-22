package server

import (
	"context"
	"path/filepath"

	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// SeedAgent describes a fixed-id agent to spawn during seeding.
type SeedAgent struct {
	ID       string
	Template string
	Name     string
}

// SeedConfig describes fixtures to install into a fresh server: one project,
// one workspace, and a set of pre-spawned agents with deterministic ids.
type SeedConfig struct {
	Project   string
	Repo      string
	Branch    string
	Templates []*arpv1.AgentTemplate
	Workspace string
	Agents    []SeedAgent
}

// adminPrincipal returns a synthetic admin principal used for internal seeding.
func adminPrincipal() *principal {
	return &principal{
		subject:        "seed",
		scope:          &arpv1.Scope{Global: true},
		perm:           arpv1.Permission_PERMISSION_ADMIN,
		localhostAdmin: true,
	}
}

// Seed installs a project, workspace and agents directly, bypassing RPC auth.
// It is intended for demo servers and for fixtures the gRPC conformance suite
// expects (e.g. echo-agent-001 / crush-agent-001 in workspace arp-test).
func (s *Server) Seed(ctx context.Context, cfg SeedConfig) error {
	branch := cfg.Branch
	if branch == "" {
		branch = "main"
	}
	s.mu.Lock()
	s.projects[cfg.Project] = &arpv1.Project{
		Name:   cfg.Project,
		Repo:   cfg.Repo,
		Branch: branch,
		Agents: cfg.Templates,
	}
	if cfg.Workspace != "" {
		if _, ok := s.workspaces[cfg.Workspace]; !ok {
			s.workspaces[cfg.Workspace] = &workspaceEntry{
				pb: &arpv1.Workspace{
					Name:      cfg.Workspace,
					Project:   cfg.Project,
					Dir:       filepath.Join(s.cfg.WorkspaceRoot, cfg.Workspace),
					Status:    arpv1.WorkspaceStatus_WORKSPACE_STATUS_ACTIVE,
					CreatedAt: nowTS(),
				},
				project: cfg.Project,
				dir:     filepath.Join(s.cfg.WorkspaceRoot, cfg.Workspace),
				agents:  map[string]bool{},
			}
		}
	}
	s.mu.Unlock()

	p := adminPrincipal()
	for _, a := range cfg.Agents {
		if _, err := s.spawnAgentInternal(ctx, p, cfg.Workspace, a.Template, a.Name, nil, nil, arpv1.Permission_PERMISSION_UNSPECIFIED, a.ID); err != nil {
			return err
		}
	}
	return nil
}
