package runner_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/aleksclark/spec-torture/internal/mockserver"
	"github.com/aleksclark/spec-torture/internal/runner"
	"github.com/aleksclark/spec-torture/internal/schema"
	"gopkg.in/yaml.v3"
)

// TestE2EAgentLifecycle runs the full ARP E2E lifecycle spec against a mock server.
// This verifies: project registration, workspace creation, agent spawning,
// status polling, A2A messaging, agent stop, workspace destruction.
func TestE2EAgentLifecycle(t *testing.T) {
	srv, err := mockserver.New()
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}
	defer srv.Close()

	// Load the E2E spec
	specData, err := os.ReadFile("../../specs/arp/spec-e2e.yaml")
	if err != nil {
		t.Fatalf("failed to read spec-e2e.yaml: %v", err)
	}

	var spec schema.Spec
	if err := yaml.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec YAML: %v", err)
	}

	if errs := spec.Validate(); len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("validation error: %v", e)
		}
		t.FailNow()
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := runner.Config{
		Runtime:        "mock",
		RuntimeVersion: "test",
		BaseURL:        srv.URL(),
		Timeout:        60 * time.Second,
	}

	r, err := runner.New(logger, cfg)
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}
	defer r.Close()

	ctx := context.Background()
	run, err := r.Run(ctx, &spec, cfg)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// Report results
	for _, result := range run.Results {
		if result.Status == schema.StatusPass {
			t.Logf("PASS: %s", result.TestCaseID)
		} else {
			t.Errorf("FAIL: %s — status=%s error=%s", result.TestCaseID, result.Status, result.ErrorMessage)
			for i, step := range result.Steps {
				if step.Status != schema.StatusPass {
					t.Errorf("  step[%d] %s: %s — %s", i, step.Action, step.Status, step.Error)
				}
			}
		}
	}

	if run.Summary.Failed > 0 || run.Summary.Errors > 0 {
		t.Errorf("E2E run: %d passed, %d failed, %d errors (compliance: %.1f%%)",
			run.Summary.Passed, run.Summary.Failed, run.Summary.Errors, run.Summary.Compliance)
	}
}

// TestE2EAgentLifecycleIndividual runs each E2E test case individually,
// providing granular pass/fail for each scenario.
func TestE2EAgentLifecycleIndividual(t *testing.T) {
	srv, err := mockserver.New()
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}
	defer srv.Close()

	specData, err := os.ReadFile("../../specs/arp/spec-e2e.yaml")
	if err != nil {
		t.Fatalf("failed to read spec-e2e.yaml: %v", err)
	}

	var spec schema.Spec
	if err := yaml.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("failed to parse spec YAML: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	for _, tc := range spec.TestCases {
		tc := tc // capture
		t.Run(tc.ID, func(t *testing.T) {
			// Each test case gets a fresh mock server to avoid state leaks
			innerSrv, err := mockserver.New()
			if err != nil {
				t.Fatalf("failed to create mock server: %v", err)
			}
			defer innerSrv.Close()

			singleSpec := schema.Spec{
				ID:        spec.ID,
				Name:      spec.Name,
				Version:   spec.Version,
				Transport: spec.Transport,
				TestCases: []schema.TestCase{tc},
			}

			cfg := runner.Config{
				Runtime:        "mock",
				RuntimeVersion: "test",
				BaseURL:        innerSrv.URL(),
				Timeout:        60 * time.Second,
			}

			r, err := runner.New(logger, cfg)
			if err != nil {
				t.Fatalf("failed to create runner: %v", err)
			}
			defer r.Close()

			ctx := context.Background()
			run, err := r.Run(ctx, &singleSpec, cfg)
			if err != nil {
				t.Fatalf("run failed: %v", err)
			}

			for _, result := range run.Results {
				if result.Status != schema.StatusPass {
					t.Errorf("test %s: status=%s error=%s", result.TestCaseID, result.Status, result.ErrorMessage)
					for i, step := range result.Steps {
						if step.Status != schema.StatusPass {
							t.Logf("  step[%d] %s: %s — %s", i, step.Action, step.Status, step.Error)
						}
					}
				}
			}
		})
	}
}

// TestMCPToolCallTranslation tests that MCP tool calls are properly translated
// to HTTP REST calls and execute correctly against the mock server.
func TestMCPToolCallTranslation(t *testing.T) {
	srv, err := mockserver.New()
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}
	defer srv.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Build a spec that uses mcp_tool_call action
	spec := schema.Spec{
		ID:        "mcp-translation-test",
		Name:      "MCP Tool Call Translation",
		Version:   "1.0.0",
		Transport: schema.TransportMCPAndHTTP,
		TestCases: []schema.TestCase{
			{
				ID:       "mcp-project-register",
				Name:     "project/register via MCP tool call",
				Severity: schema.SeverityRequired,
				Category: "mcp-translation",
				Timeout:  10 * time.Second,
				Steps: []schema.Step{
					{
						Action: schema.ActionMCPToolCall,
						Payload: map[string]any{
							"tool": "project/register",
							"arguments": map[string]any{
								"name": "mcp-test-project",
								"repo": "/tmp/mcp-test-repo",
							},
						},
					},
					{
						Action: schema.ActionExpect,
						Expect: map[string]any{
							"http_status": 201,
							"body": map[string]any{
								"name": "mcp-test-project",
							},
						},
					},
				},
			},
			{
				ID:       "mcp-workspace-create",
				Name:     "workspace/create via MCP tool call",
				Severity: schema.SeverityRequired,
				Category: "mcp-translation",
				Timeout:  10 * time.Second,
				Steps: []schema.Step{
					// First register a project (needed for workspace)
					{
						Action: schema.ActionMCPToolCall,
						Payload: map[string]any{
							"tool": "project/register",
							"arguments": map[string]any{
								"name": "mcp-ws-project",
								"repo": "/tmp/mcp-ws-repo",
							},
						},
					},
					{
						Action: schema.ActionExpect,
						Expect: map[string]any{
							"http_status": 201,
						},
					},
					// Create workspace
					{
						Action: schema.ActionMCPToolCall,
						Payload: map[string]any{
							"tool": "workspace/create",
							"arguments": map[string]any{
								"name":    "mcp-ws-test",
								"project": "mcp-ws-project",
							},
						},
					},
					{
						Action: schema.ActionExpect,
						Expect: map[string]any{
							"http_status": 201,
							"body": map[string]any{
								"name":    "mcp-ws-test",
								"project": "mcp-ws-project",
								"status":  "active",
							},
						},
					},
				},
			},
			{
				ID:       "mcp-agent-spawn-and-status",
				Name:     "agent/spawn and agent/status via MCP tool call",
				Severity: schema.SeverityRequired,
				Category: "mcp-translation",
				Timeout:  30 * time.Second,
				Steps: []schema.Step{
					{
						Action: schema.ActionMCPToolCall,
						Payload: map[string]any{
							"tool": "project/register",
							"arguments": map[string]any{
								"name": "mcp-agent-project",
								"repo": "/tmp/mcp-agent-repo",
								"agents": []any{
									map[string]any{
										"name":     "test-agent",
										"command":  "echo serve",
										"port_env": "A2A_PORT",
									},
								},
							},
						},
					},
					{
						Action: schema.ActionExpect,
						Expect: map[string]any{
							"http_status": 201,
						},
					},
					{
						Action: schema.ActionMCPToolCall,
						Payload: map[string]any{
							"tool": "workspace/create",
							"arguments": map[string]any{
								"name":    "mcp-agent-ws",
								"project": "mcp-agent-project",
							},
						},
					},
					{
						Action: schema.ActionExpect,
						Expect: map[string]any{
							"http_status": 201,
						},
					},
					{
						Action: schema.ActionMCPToolCall,
						Payload: map[string]any{
							"tool": "agent/spawn",
							"arguments": map[string]any{
								"workspace": "mcp-agent-ws",
								"template":  "test-agent",
							},
						},
					},
					{
						Action: schema.ActionExpect,
						Expect: map[string]any{
							"http_status": 201,
							"body": map[string]any{
								"id":        "*",
								"template":  "test-agent",
								"workspace": "mcp-agent-ws",
							},
						},
					},
				},
			},
		},
	}

	cfg := runner.Config{
		Runtime:        "mock",
		RuntimeVersion: "test",
		BaseURL:        srv.URL(),
		Timeout:        30 * time.Second,
	}

	r, err := runner.New(logger, cfg)
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}
	defer r.Close()

	ctx := context.Background()
	run, err := r.Run(ctx, &spec, cfg)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	for _, result := range run.Results {
		if result.Status != schema.StatusPass {
			t.Errorf("FAIL: %s — status=%s error=%s", result.TestCaseID, result.Status, result.ErrorMessage)
			for i, step := range result.Steps {
				if step.Status != schema.StatusPass {
					t.Errorf("  step[%d] %s: %s — %s", i, step.Action, step.Status, step.Error)
				}
			}
		} else {
			t.Logf("PASS: %s", result.TestCaseID)
		}
	}
}
