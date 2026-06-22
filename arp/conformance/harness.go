// Package conformance is an executable conformance suite for the Agent Registry
// Protocol (ARP). It boots the reference server with an in-process mock agent
// backend, drives it through the reference client, and verifies every normative
// requirement from arp-spec/: gRPC status codes, the agent lifecycle state
// machine, AgentCard enrichment, discovery filtering, server-streaming watches,
// and the token scope/permission/session model.
package conformance

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.com/aleksclark/spec-torture/arp/backend"
	"github.com/aleksclark/spec-torture/arp/client"
	"github.com/aleksclark/spec-torture/arp/server"
	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// Severity levels mirror the spec's RFC 2119 mapping.
const (
	Required    = "required"
	Recommended = "recommended"
	Optional    = "optional"
)

// Result is the outcome of a single conformance check.
type Result struct {
	Group    string
	ID       string
	Severity string
	Pass     bool
	Detail   string
}

// Env is a running reference server plus helpers to connect clients.
type Env struct {
	Addr    string
	Server  *server.Server
	backend *backend.MockBackend
	gs      *grpc.Server
	lis     net.Listener
}

// Start boots an in-process reference server with demo fixtures and no gateway
// (the clean control-plane-only deployment: agents are reached via direct_url).
func Start(ctx context.Context) (*Env, error) {
	return start(ctx, "")
}

// StartWithGateway boots a reference server configured with an optional A2A
// gateway base URL, so agents advertise a proxy_url. Used to exercise the
// optional gateway profile's effect on AgentCard enrichment.
func StartWithGateway(ctx context.Context, gatewayURL string) (*Env, error) {
	return start(ctx, gatewayURL)
}

func start(ctx context.Context, gatewayURL string) (*Env, error) {
	mock := backend.NewMockBackend()

	srv := server.New(server.Config{
		GatewayBaseURL: gatewayURL,
		PortRange:      server.PortRange{Min: 9100, Max: 9399},
		LocalhostAdmin: true,
		Backend:        mock,
	})
	if err := srv.Seed(ctx, demoSeed()); err != nil {
		return nil, fmt.Errorf("seed: %w", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	gs := grpc.NewServer()
	srv.Register(gs)
	go func() { _ = gs.Serve(lis) }()

	return &Env{
		Addr:    lis.Addr().String(),
		Server:  srv,
		backend: mock,
		gs:      gs,
		lis:     lis,
	}, nil
}

// Stop tears down the server and all mock agents.
func (e *Env) Stop() {
	e.gs.Stop()
	e.Server.Stop()
	e.backend.Close()
}

// Admin returns a client with no token (localhost admin).
func (e *Env) Admin() (*client.Client, error) {
	return client.Dial(e.Addr)
}

// TokenClient returns a client authenticated with the given bearer token.
func (e *Env) TokenClient(token string) (*client.Client, error) {
	return client.Dial(e.Addr, client.WithToken(token))
}

// Summary aggregates results into compliance figures.
type Summary struct {
	Total      int
	Passed     int
	Failed     int
	Skipped    int
	Compliance float64
}

// Summarize computes a Summary from results (optional failures don't count).
func Summarize(results []Result) Summary {
	var s Summary
	for _, r := range results {
		s.Total++
		if r.Pass {
			s.Passed++
		} else {
			s.Failed++
		}
	}
	denom := s.Total
	if denom == 0 {
		return s
	}
	s.Compliance = float64(s.Passed) / float64(denom) * 100
	return s
}

// RequiredFailures returns the ids of failing required checks.
func RequiredFailures(results []Result) []string {
	var out []string
	for _, r := range results {
		if !r.Pass && r.Severity == Required {
			out = append(out, r.ID)
		}
	}
	return out
}

// Markdown renders results in the repository's gRPC report format.
func Markdown(title string, results []Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)

	groups := []string{}
	byGroup := map[string][]Result{}
	for _, r := range results {
		if _, ok := byGroup[r.Group]; !ok {
			groups = append(groups, r.Group)
		}
		byGroup[r.Group] = append(byGroup[r.Group], r)
	}
	for _, g := range groups {
		fmt.Fprintf(&b, "## %s\n", g)
		for _, r := range byGroup[g] {
			status := "PASS"
			if !r.Pass {
				status = "FAIL"
			}
			line := fmt.Sprintf("  %s  %-52s %s", status, r.ID, r.Severity)
			if !r.Pass && r.Detail != "" {
				line += "  " + r.Detail
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	s := Summarize(results)
	b.WriteString("## Summary\n\n")
	b.WriteString("| Metric | Count |\n|--------|-------|\n")
	fmt.Fprintf(&b, "| Total | %d |\n", s.Total)
	fmt.Fprintf(&b, "| Passed | %d |\n", s.Passed)
	fmt.Fprintf(&b, "| Failed | %d |\n", s.Failed)
	fmt.Fprintf(&b, "| **Compliance** | **%.1f%%** |\n", s.Compliance)

	if fails := failingLines(results); len(fails) > 0 {
		b.WriteString("\n## Failures\n\n")
		for _, f := range fails {
			b.WriteString("  " + f + "\n")
		}
	}
	return b.String()
}

func failingLines(results []Result) []string {
	var out []string
	for _, r := range results {
		if !r.Pass {
			out = append(out, fmt.Sprintf("FAIL  %s (%s): %s", r.ID, r.Severity, r.Detail))
		}
	}
	sort.Strings(out)
	return out
}

func demoSeed() server.SeedConfig {
	return server.SeedConfig{
		Project: "myapp",
		Repo:    "/tmp/myapp",
		Branch:  "main",
		Templates: []*arpv1.AgentTemplate{
			{
				Name:    "crush",
				Command: "crush serve",
				PortEnv: "A2A_PORT",
				A2ACardConfig: &arpv1.A2ACardConfig{
					Name:        "Crush",
					Description: "AI coding assistant",
					Streaming:   true,
					Skills: []*arpv1.AgentSkillConfig{{
						Id:          "code",
						Name:        "Code",
						Description: "Write, review, and debug code",
						Tags:        []string{"crush", "coding", "code"},
					}},
				},
			},
			{
				Name:    "echo",
				Command: "echo serve",
				PortEnv: "A2A_PORT",
				A2ACardConfig: &arpv1.A2ACardConfig{
					Name:        "Echo",
					Description: "Echoes input",
					Skills: []*arpv1.AgentSkillConfig{{
						Id:          "echo",
						Name:        "Echo",
						Description: "Echo skill",
						Tags:        []string{"echo"},
					}},
				},
			},
		},
		Workspace: "arp-test",
		Agents: []server.SeedAgent{
			{ID: "echo-agent-001", Template: "echo", Name: "echo-agent"},
			{ID: "crush-agent-001", Template: "crush", Name: "crush-agent"},
		},
	}
}

// shortCtx returns a context with a default timeout for checks.
func shortCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}
