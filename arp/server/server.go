// Package server provides a reference implementation of the Agent Registry
// Protocol (ARP) v0.4 gRPC services: ProjectService, WorkspaceService,
// AgentService, DiscoveryService and TokenService.
//
// The implementation is in-memory and backend-agnostic: agent processes are
// started through a backend.Backend, and agent communication is proxied over
// A2A v1.0 HTTP+JSON. It enforces the token scope/permission/session model
// described in arp-spec/identity-and-scopes.md and returns the gRPC status
// codes mandated by the per-service spec documents.
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/aleksclark/spec-torture/arp/a2a"
	"github.com/aleksclark/spec-torture/arp/backend"
	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
	a2av1 "github.com/aleksclark/spec-torture/gen/lf/a2a/v1"
)

// PortRange is the inclusive range of ports the server allocates to agents.
type PortRange struct {
	Min int
	Max int
}

// Config configures a reference ARP server.
type Config struct {
	// GatewayBaseURL is the externally reachable base URL of an optional A2A
	// gateway deployed in front of agents (see arp-spec/profile-a2a-gateway.md).
	// When set, agents' proxy_url is populated as
	// "{GatewayBaseURL}/a2a/agents/{id}". Leave empty (the default) for the
	// clean control-plane-only deployment: clients reach agents via direct_url.
	GatewayBaseURL string
	// PortRange bounds agent port allocation. Defaults to 9100-9199.
	PortRange PortRange
	// LocalhostAdmin treats unauthenticated loopback callers as ADMIN with
	// global scope. Defaults to true.
	LocalhostAdmin bool
	// Backend starts agent processes. Defaults to backend.NewExecBackend().
	Backend backend.Backend
	// Templates are globally available agent templates, merged with per-project
	// templates when spawning.
	Templates map[string]*arpv1.AgentTemplate
	// WorkspaceRoot is the base directory under which workspace dirs are placed.
	WorkspaceRoot string
	// A2ATimeout bounds the AgentCard discovery/health requests ARP makes to
	// agents. Defaults to 30s.
	A2ATimeout time.Duration
}

func (c *Config) applyDefaults() {
	c.GatewayBaseURL = strings.TrimRight(c.GatewayBaseURL, "/")
	if c.PortRange.Min == 0 && c.PortRange.Max == 0 {
		c.PortRange = PortRange{Min: 9100, Max: 9199}
	}
	if c.Backend == nil {
		c.Backend = backend.NewExecBackend()
	}
	if c.A2ATimeout <= 0 {
		c.A2ATimeout = 30 * time.Second
	}
	if c.WorkspaceRoot == "" {
		c.WorkspaceRoot = "/tmp/arp-workspaces"
	}
}

// Server is the reference ARP server. It implements all five ARP gRPC services.
type Server struct {
	arpv1.UnimplementedProjectServiceServer
	arpv1.UnimplementedWorkspaceServiceServer
	arpv1.UnimplementedAgentServiceServer
	arpv1.UnimplementedDiscoveryServiceServer
	arpv1.UnimplementedTokenServiceServer

	cfg Config
	a2a *a2a.Client

	mu         sync.Mutex
	projects   map[string]*arpv1.Project
	workspaces map[string]*workspaceEntry
	agents     map[string]*agentEntry
	tokens     map[string]*tokenEntry // by id
	byBearer   map[string]*tokenEntry
	usedPorts  map[int]bool

	subSeq    int
	agentSubs map[string]map[int]chan *arpv1.AgentEvent
	wsSubs    map[string]map[int]chan *arpv1.WorkspaceEvent
}

type workspaceEntry struct {
	pb      *arpv1.Workspace
	project string
	dir     string
	agents  map[string]bool // agent ids
}

type agentEntry struct {
	pb       *arpv1.AgentInstance
	handle   backend.Handle
	template *arpv1.AgentTemplate
	project  string
	name     string
	env      map[string]string
	cardBase map[string]any
}

type tokenEntry struct {
	id            string
	subject       string
	scope         *arpv1.Scope
	perm          arpv1.Permission
	sessionID     string
	parentTokenID string
	bearer        string
	issuedAt      time.Time
	expiresAt     time.Time
}

// New creates a reference ARP server.
func New(cfg Config) *Server {
	cfg.applyDefaults()
	return &Server{
		cfg:        cfg,
		a2a:        a2a.NewClient(cfg.A2ATimeout),
		projects:   map[string]*arpv1.Project{},
		workspaces: map[string]*workspaceEntry{},
		agents:     map[string]*agentEntry{},
		tokens:     map[string]*tokenEntry{},
		byBearer:   map[string]*tokenEntry{},
		usedPorts:  map[int]bool{},
		agentSubs:  map[string]map[int]chan *arpv1.AgentEvent{},
		wsSubs:     map[string]map[int]chan *arpv1.WorkspaceEvent{},
	}
}

// Register registers all ARP services on the given gRPC server.
func (s *Server) Register(gs *grpc.Server) {
	arpv1.RegisterProjectServiceServer(gs, s)
	arpv1.RegisterWorkspaceServiceServer(gs, s)
	arpv1.RegisterAgentServiceServer(gs, s)
	arpv1.RegisterDiscoveryServiceServer(gs, s)
	arpv1.RegisterTokenServiceServer(gs, s)
}

// Stop terminates all running agents.
func (s *Server) Stop() {
	s.mu.Lock()
	handles := make([]backend.Handle, 0, len(s.agents))
	for _, a := range s.agents {
		if a.handle != nil {
			handles = append(handles, a.handle)
		}
	}
	s.mu.Unlock()
	for _, h := range handles {
		_ = h.Stop(context.Background(), 1000)
	}
}

// ---------------------------------------------------------------------------
// Authentication & principals
// ---------------------------------------------------------------------------

type principal struct {
	token          *tokenEntry
	subject        string
	scope          *arpv1.Scope // nil => global
	perm           arpv1.Permission
	sessionID      string
	localhostAdmin bool
}

func (p *principal) isGlobal() bool {
	return p.scope == nil || p.scope.GetGlobal()
}

func (p *principal) canAccessProject(name string) bool {
	if p.isGlobal() {
		return true
	}
	for _, pr := range p.scope.GetProjects() {
		if pr == name {
			return true
		}
	}
	return false
}

// authenticate resolves the caller principal from gRPC metadata. An explicit
// Bearer token always takes precedence; otherwise loopback callers may be
// granted admin when LocalhostAdmin is enabled.
func (s *Server) authenticate(ctx context.Context) (*principal, error) {
	if bearer := bearerFromContext(ctx); bearer != "" {
		s.mu.Lock()
		tok, ok := s.byBearer[bearer]
		s.mu.Unlock()
		if !ok {
			return nil, errUnauthenticated("unknown bearer token")
		}
		if !tok.expiresAt.IsZero() && time.Now().After(tok.expiresAt) {
			return nil, errUnauthenticated("token expired")
		}
		return &principal{
			token:     tok,
			subject:   tok.subject,
			scope:     tok.scope,
			perm:      tok.perm,
			sessionID: tok.sessionID,
		}, nil
	}
	if s.cfg.LocalhostAdmin && isLoopback(ctx) {
		return &principal{
			subject:        "localhost",
			scope:          &arpv1.Scope{Global: true},
			perm:           arpv1.Permission_PERMISSION_ADMIN,
			localhostAdmin: true,
		}, nil
	}
	return nil, errUnauthenticated("missing authorization token")
}

func bearerFromContext(ctx context.Context) string {
	md, ok := metadataFromContext(ctx)
	if !ok {
		return ""
	}
	for _, key := range []string{"authorization", "Authorization"} {
		for _, v := range md[strings.ToLower(key)] {
			if len(v) > 7 && strings.EqualFold(v[:7], "Bearer ") {
				return strings.TrimSpace(v[7:])
			}
		}
	}
	return ""
}

func isLoopback(ctx context.Context) bool {
	pr, ok := peer.FromContext(ctx)
	if !ok || pr.Addr == nil {
		// No peer info (e.g. in-process): treat as loopback.
		return true
	}
	host, _, err := net.SplitHostPort(pr.Addr.String())
	if err != nil {
		host = pr.Addr.String()
	}
	if host == "bufconn" || host == "" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ---------------------------------------------------------------------------
// Visibility helpers (scope + session enforcement)
// ---------------------------------------------------------------------------

// canSeeAgent reports whether the principal may see/act on the agent under the
// scope+session model. Caller must hold s.mu.
func (s *Server) canSeeAgent(p *principal, a *agentEntry) bool {
	if !p.canAccessProject(a.project) {
		return false
	}
	if p.perm == arpv1.Permission_PERMISSION_SESSION {
		return p.sessionID != "" && a.pb.GetSessionId() == p.sessionID
	}
	return true
}

// workspaceVisible reports whether the principal may see the workspace. Caller
// must hold s.mu.
func (s *Server) workspaceVisible(p *principal, w *workspaceEntry) bool {
	if !p.canAccessProject(w.project) {
		return false
	}
	if p.perm == arpv1.Permission_PERMISSION_SESSION {
		if p.sessionID == "" {
			return false
		}
		for id := range w.agents {
			if a := s.agents[id]; a != nil && a.pb.GetSessionId() == p.sessionID {
				return true
			}
		}
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Ports, ids, sessions
// ---------------------------------------------------------------------------

// allocPort reserves a free port within the configured range. Caller must hold
// s.mu.
func (s *Server) allocPort() (int, error) {
	for port := s.cfg.PortRange.Min; port <= s.cfg.PortRange.Max; port++ {
		if s.usedPorts[port] {
			continue
		}
		if portFree(port) {
			s.usedPorts[port] = true
			return port, nil
		}
	}
	// Fall back to an OS-assigned port outside the range.
	p, err := backend.FreePort()
	if err != nil {
		return 0, err
	}
	s.usedPorts[p] = true
	return p, nil
}

func (s *Server) freePort(port int) {
	delete(s.usedPorts, port)
}

func portFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func newID(prefix string) string {
	return prefix + "-" + strings.Split(uuid.NewString(), "-")[0]
}

func randomToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "arp_" + hex.EncodeToString(b)
}

// ---------------------------------------------------------------------------
// Event publication
// ---------------------------------------------------------------------------

func (s *Server) subscribeAgent(agentID string) (int, chan *arpv1.AgentEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subSeq++
	id := s.subSeq
	ch := make(chan *arpv1.AgentEvent, 32)
	if s.agentSubs[agentID] == nil {
		s.agentSubs[agentID] = map[int]chan *arpv1.AgentEvent{}
	}
	s.agentSubs[agentID][id] = ch
	return id, ch
}

func (s *Server) unsubscribeAgent(agentID string, id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m := s.agentSubs[agentID]; m != nil {
		delete(m, id)
	}
}

func (s *Server) subscribeWorkspace(ws string) (int, chan *arpv1.WorkspaceEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subSeq++
	id := s.subSeq
	ch := make(chan *arpv1.WorkspaceEvent, 32)
	if s.wsSubs[ws] == nil {
		s.wsSubs[ws] = map[int]chan *arpv1.WorkspaceEvent{}
	}
	s.wsSubs[ws][id] = ch
	return id, ch
}

func (s *Server) unsubscribeWorkspace(ws string, id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m := s.wsSubs[ws]; m != nil {
		delete(m, id)
	}
}

// publishAgentEvent emits an agent event and a mirrored workspace event. Caller
// must NOT hold s.mu.
func (s *Server) publishAgentEvent(a *agentEntry, evType arpv1.AgentEventType, card *a2av1.AgentCard) {
	agentPB := cloneAgent(a.pb)
	ev := &arpv1.AgentEvent{EventType: evType, Agent: agentPB, AgentCard: card}
	wsEvType := arpv1.WorkspaceEventType_WORKSPACE_EVENT_TYPE_AGENT_STATUS_CHANGED
	switch evType {
	case arpv1.AgentEventType_AGENT_EVENT_TYPE_STOPPED:
		wsEvType = arpv1.WorkspaceEventType_WORKSPACE_EVENT_TYPE_AGENT_STOPPED
	}
	s.mu.Lock()
	for _, ch := range s.agentSubs[a.pb.GetId()] {
		trySend(ch, ev)
	}
	ws := s.workspaces[a.pb.GetWorkspace()]
	var wsPB *arpv1.Workspace
	if ws != nil {
		wsPB = s.buildWorkspacePB(ws)
	}
	subs := s.wsSubs[a.pb.GetWorkspace()]
	s.mu.Unlock()
	if wsPB != nil {
		wsEv := &arpv1.WorkspaceEvent{EventType: wsEvType, Workspace: wsPB, Agent: agentPB}
		s.mu.Lock()
		for _, ch := range subs {
			trySend(ch, wsEv)
		}
		s.mu.Unlock()
	}
}

func (s *Server) publishWorkspaceSpawn(ws *workspaceEntry, a *agentEntry) {
	s.mu.Lock()
	wsPB := s.buildWorkspacePB(ws)
	subs := s.wsSubs[ws.pb.GetName()]
	ev := &arpv1.WorkspaceEvent{
		EventType: arpv1.WorkspaceEventType_WORKSPACE_EVENT_TYPE_AGENT_SPAWNED,
		Workspace: wsPB,
		Agent:     cloneAgent(a.pb),
	}
	for _, ch := range subs {
		trySend(ch, ev)
	}
	s.mu.Unlock()
}

func (s *Server) publishWorkspaceDestroyed(ws *workspaceEntry) {
	s.mu.Lock()
	wsPB := s.buildWorkspacePB(ws)
	subs := s.wsSubs[ws.pb.GetName()]
	ev := &arpv1.WorkspaceEvent{
		EventType: arpv1.WorkspaceEventType_WORKSPACE_EVENT_TYPE_WORKSPACE_DESTROYED,
		Workspace: wsPB,
	}
	for _, ch := range subs {
		trySend(ch, ev)
	}
	s.mu.Unlock()
}

func trySend[T any](ch chan T, v T) {
	select {
	case ch <- v:
	default:
	}
}

// ---------------------------------------------------------------------------
// Protobuf builders
// ---------------------------------------------------------------------------

// buildWorkspacePB returns a Workspace message including its agents. Caller
// must hold s.mu.
func (s *Server) buildWorkspacePB(w *workspaceEntry) *arpv1.Workspace {
	pb := cloneWorkspace(w.pb)
	ids := make([]string, 0, len(w.agents))
	for id := range w.agents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	pb.Agents = nil
	for _, id := range ids {
		if a := s.agents[id]; a != nil {
			pb.Agents = append(pb.Agents, cloneAgent(a.pb))
		}
	}
	return pb
}

func agentStatusString(st arpv1.AgentStatus) string {
	return strings.ToLower(strings.TrimPrefix(st.String(), "AGENT_STATUS_"))
}

// enrichedCard builds the ARP-enriched A2A AgentCard for an agent: it converts
// the agent's own card to a typed lf.a2a.v1.AgentCard, sets supported_interfaces
// to the canonical direct_url (plus the gateway URL when one is configured), and
// injects the metadata.arp lifecycle block. Caller must hold s.mu.
func (s *Server) enrichedCard(a *agentEntry) *a2av1.AgentCard {
	card, err := mapToAgentCard(a.cardBase)
	if err != nil || card == nil {
		card = &a2av1.AgentCard{}
	}
	// Direct is the canonical interface: clients reach the agent here.
	ifaces := []*a2av1.AgentInterface{
		{Url: a.pb.GetDirectUrl(), Transport: "HTTP_JSON"},
	}
	if a.pb.GetProxyUrl() != "" {
		ifaces = append(ifaces, &a2av1.AgentInterface{Url: a.pb.GetProxyUrl(), Transport: "HTTP_JSON"})
	}
	card.SupportedInterfaces = ifaces

	arpMeta := map[string]any{
		"agent_id":   a.pb.GetId(),
		"workspace":  a.pb.GetWorkspace(),
		"project":    a.project,
		"template":   a.pb.GetTemplate(),
		"status":     agentStatusString(a.pb.GetStatus()),
		"direct_url": a.pb.GetDirectUrl(),
	}
	if a.pb.GetProxyUrl() != "" {
		arpMeta["proxy_url"] = a.pb.GetProxyUrl()
	}
	if a.pb.GetStartedAt() != nil {
		arpMeta["started_at"] = a.pb.GetStartedAt().AsTime().UTC().Format(time.RFC3339)
	}
	meta := card.GetMetadata().AsMap()
	if meta == nil {
		meta = map[string]any{}
	}
	meta["arp"] = arpMeta
	if st, err := structpb.NewStruct(meta); err == nil {
		card.Metadata = st
	}
	return card
}

// refreshCard recomputes and stores the agent's enriched card. Caller must hold
// s.mu.
func (s *Server) refreshCard(a *agentEntry) {
	a.pb.A2AAgentCard = s.enrichedCard(a)
}

func nowTS() *timestamppb.Timestamp { return timestamppb.New(time.Now().UTC()) }
