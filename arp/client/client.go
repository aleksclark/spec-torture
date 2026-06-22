// Package client is a reference client library for the Agent Registry Protocol
// (ARP). It wraps the generated gRPC service clients, injects bearer-token
// authentication via gRPC metadata, and offers ergonomic helpers for the most
// common operations.
package client

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// Client is a connected ARP client exposing all five ARP services.
type Client struct {
	conn *grpc.ClientConn

	Project   arpv1.ProjectServiceClient
	Workspace arpv1.WorkspaceServiceClient
	Agent     arpv1.AgentServiceClient
	Discovery arpv1.DiscoveryServiceClient
	Token     arpv1.TokenServiceClient

	token string
}

type options struct {
	token    string
	dialOpts []grpc.DialOption
}

// Option configures a Client.
type Option func(*options)

// WithToken sets the bearer token attached to every request as
// "authorization: Bearer <token>" gRPC metadata.
func WithToken(token string) Option {
	return func(o *options) { o.token = token }
}

// WithDialOption appends raw gRPC dial options.
func WithDialOption(opts ...grpc.DialOption) Option {
	return func(o *options) { o.dialOpts = append(o.dialOpts, opts...) }
}

// Dial connects to an ARP server at target (host:port) over plaintext h2.
func Dial(target string, opts ...Option) (*Client, error) {
	var o options
	for _, fn := range opts {
		fn(&o)
	}
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if o.token != "" {
		dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(bearerCreds{token: o.token}))
	}
	dialOpts = append(dialOpts, o.dialOpts...)

	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:      conn,
		Project:   arpv1.NewProjectServiceClient(conn),
		Workspace: arpv1.NewWorkspaceServiceClient(conn),
		Agent:     arpv1.NewAgentServiceClient(conn),
		Discovery: arpv1.NewDiscoveryServiceClient(conn),
		Token:     arpv1.NewTokenServiceClient(conn),
		token:     o.token,
	}, nil
}

// Conn returns the underlying gRPC connection.
func (c *Client) Conn() *grpc.ClientConn { return c.conn }

// Close releases the connection.
func (c *Client) Close() error { return c.conn.Close() }

// ---------------------------------------------------------------------------
// Convenience helpers
// ---------------------------------------------------------------------------

// ListProjects lists projects visible to the client's token.
func (c *Client) ListProjects(ctx context.Context) ([]*arpv1.Project, error) {
	resp, err := c.Project.ListProjects(ctx, &arpv1.ListProjectsRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetProjects(), nil
}

// RegisterProject registers a project with optional templates.
func (c *Client) RegisterProject(ctx context.Context, name, repo string, agents ...*arpv1.AgentTemplate) (*arpv1.Project, error) {
	return c.Project.RegisterProject(ctx, &arpv1.RegisterProjectRequest{
		Name:   name,
		Repo:   repo,
		Agents: agents,
	})
}

// UnregisterProject removes a project.
func (c *Client) UnregisterProject(ctx context.Context, name string) error {
	_, err := c.Project.UnregisterProject(ctx, &arpv1.UnregisterProjectRequest{Name: name})
	return err
}

// CreateWorkspace creates a workspace, optionally auto-spawning templates.
func (c *Client) CreateWorkspace(ctx context.Context, name, project string, autoAgents ...string) (*arpv1.Workspace, error) {
	return c.Workspace.CreateWorkspace(ctx, &arpv1.CreateWorkspaceRequest{
		Name:       name,
		Project:    project,
		AutoAgents: autoAgents,
	})
}

// ListWorkspaces lists visible workspaces.
func (c *Client) ListWorkspaces(ctx context.Context) ([]*arpv1.Workspace, error) {
	resp, err := c.Workspace.ListWorkspaces(ctx, &arpv1.ListWorkspacesRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetWorkspaces(), nil
}

// GetWorkspace fetches a workspace by name.
func (c *Client) GetWorkspace(ctx context.Context, name string) (*arpv1.Workspace, error) {
	return c.Workspace.GetWorkspace(ctx, &arpv1.GetWorkspaceRequest{Name: name})
}

// DestroyWorkspace destroys a workspace.
func (c *Client) DestroyWorkspace(ctx context.Context, name string) error {
	_, err := c.Workspace.DestroyWorkspace(ctx, &arpv1.DestroyWorkspaceRequest{Name: name})
	return err
}

// SpawnAgent spawns an agent from a template in a workspace.
func (c *Client) SpawnAgent(ctx context.Context, workspace, template, name string) (*arpv1.AgentInstance, error) {
	return c.Agent.SpawnAgent(ctx, &arpv1.SpawnAgentRequest{
		Workspace: workspace,
		Template:  template,
		Name:      name,
	})
}

// ListAgents lists visible agents.
func (c *Client) ListAgents(ctx context.Context) ([]*arpv1.AgentInstance, error) {
	resp, err := c.Agent.ListAgents(ctx, &arpv1.ListAgentsRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetAgents(), nil
}

// GetAgentStatus fetches a single agent.
func (c *Client) GetAgentStatus(ctx context.Context, agentID string) (*arpv1.AgentInstance, error) {
	return c.Agent.GetAgentStatus(ctx, &arpv1.GetAgentStatusRequest{AgentId: agentID})
}

// DirectURL returns the agent's A2A v1.0 endpoint. ARP does not relay messages;
// callers send A2A SendMessage / GetTask to this URL directly using any A2A
// client. This is the seam between ARP (lifecycle/registry) and A2A (messaging).
func (c *Client) DirectURL(ctx context.Context, agentID string) (string, error) {
	a, err := c.GetAgentStatus(ctx, agentID)
	if err != nil {
		return "", err
	}
	return a.GetDirectUrl(), nil
}

// StopAgent stops an agent.
func (c *Client) StopAgent(ctx context.Context, agentID string) (*arpv1.AgentInstance, error) {
	return c.Agent.StopAgent(ctx, &arpv1.StopAgentRequest{AgentId: agentID})
}

// DiscoverAgents discovers agents locally, optionally filtered by capability.
func (c *Client) DiscoverAgents(ctx context.Context, capability string) (*arpv1.DiscoverAgentsResponse, error) {
	return c.Discovery.DiscoverAgents(ctx, &arpv1.DiscoverAgentsRequest{
		Scope:      arpv1.DiscoveryScope_DISCOVERY_SCOPE_LOCAL,
		Capability: capability,
	})
}

// CreateToken issues a token. Requires admin permission on the caller.
func (c *Client) CreateToken(ctx context.Context, subject string, scope *arpv1.Scope, perm arpv1.Permission, ttl time.Duration) (*arpv1.CreateTokenResponse, error) {
	return c.Token.CreateToken(ctx, &arpv1.CreateTokenRequest{
		Subject:          subject,
		Scope:            scope,
		Permission:       perm,
		ExpiresInSeconds: int32(ttl / time.Second),
	})
}

// bearerCreds implements credentials.PerRPCCredentials for plaintext bearer
// token injection.
type bearerCreds struct{ token string }

func (b bearerCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + b.token}, nil
}

func (bearerCreds) RequireTransportSecurity() bool { return false }
