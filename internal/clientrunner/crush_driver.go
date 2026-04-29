package clientrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CrushDriver drives crush-a2a's A2A client tools via `crush run`.
type CrushDriver struct {
	crushBin string
	homeDir  string
}

// NewCrushDriver creates a driver that invokes the crush binary's A2A tools.
func NewCrushDriver(crushBin, homeDir string) *CrushDriver {
	return &CrushDriver{
		crushBin: crushBin,
		homeDir:  homeDir,
	}
}

// ConfigureMockServer writes a crush.json that registers the mock as an A2A server.
func (d *CrushDriver) ConfigureMockServer(mockURL string) error {
	configDir := filepath.Join(d.homeDir, ".config", "crush")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	config := map[string]any{
		"options": map[string]any{
			"plugins": map[string]any{
				"a2a": map[string]any{
					"servers": []any{
						map[string]any{
							"name": "mock-agent",
							"url":  mockURL,
						},
					},
				},
			},
			"disabled_plugins": []string{"otlp", "agent-status", "periodic-prompts", "tempotown", "a2a-server"},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(configDir, "crush.json"), data, 0o644)
}

func (d *CrushDriver) Discovery(ctx context.Context, agentURL string) (map[string]any, error) {
	return d.runCrush(ctx,
		`Use the a2a_list_agents tool to list all available A2A agents. Return only the JSON result.`,
	)
}

func (d *CrushDriver) SendMessage(ctx context.Context, agentURL string, params map[string]any) (map[string]any, error) {
	text, _ := params["text"].(string)
	if text == "" {
		text = "Hello"
	}

	return d.runCrush(ctx, fmt.Sprintf(
		`Use the a2a_send_message tool to send a message to the "mock-agent" A2A server. `+
			`The message text should be: %q. Return only the JSON result.`,
		text,
	))
}

func (d *CrushDriver) GetTask(ctx context.Context, agentURL string, taskID string) (map[string]any, error) {
	return d.runCrush(ctx, fmt.Sprintf(
		`Use the a2a_get_task tool to get task %q from the "mock-agent" A2A server. Return only the JSON result.`,
		taskID,
	))
}

func (d *CrushDriver) SendStreamingMessage(ctx context.Context, agentURL string, params map[string]any) (map[string]any, error) {
	text, _ := params["text"].(string)
	if text == "" {
		text = "Hello"
	}

	return d.runCrush(ctx, fmt.Sprintf(
		`Use the a2a_send_message tool with streaming enabled to send a message to the "mock-agent" A2A server. `+
			`The message text should be: %q. Return only the JSON result.`,
		text,
	))
}

func (d *CrushDriver) Close() error {
	return nil
}

func (d *CrushDriver) runCrush(ctx context.Context, prompt string) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, d.crushBin, "run", prompt)
	cmd.Env = append(cmd.Environ(),
		fmt.Sprintf("HOME=%s", d.homeDir),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("crush run failed: %w\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	result := extractJSON(output)
	if result != nil {
		return result, nil
	}

	return map[string]any{"raw_output": output}, nil
}

func extractJSON(s string) map[string]any {
	start := strings.Index(s, "{")
	if start < 0 {
		return nil
	}

	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				var result map[string]any
				if err := json.Unmarshal([]byte(s[start:i+1]), &result); err == nil {
					return result
				}
			}
		}
	}
	return nil
}
