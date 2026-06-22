// Package a2a provides a minimal A2A v1.0 HTTP+JSON client used by the ARP
// reference server for AgentCard discovery and readiness health checks.
//
// ARP is a control plane: it discovers and reports on A2A agents but does NOT
// relay agent-to-agent messages. This client therefore only fetches AgentCards
// and probes health — it deliberately has no SendMessage/GetTask methods.
// Clients send messages to agents directly via the agent's direct_url with a
// full A2A client of their choosing.
package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an A2A v1.0 HTTP+JSON discovery/health client.
type Client struct {
	http *http.Client
}

// NewClient returns an A2A client with the given per-request timeout.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{http: &http.Client{Timeout: timeout}}
}

// ErrNotFound is returned when the agent responds with HTTP 404.
type ErrNotFound struct{ Path string }

func (e *ErrNotFound) Error() string { return "a2a: not found: " + e.Path }

// ErrAgent wraps a non-2xx, non-404 agent response.
type ErrAgent struct {
	Status int
	Body   string
}

func (e *ErrAgent) Error() string {
	return fmt.Sprintf("a2a: agent returned HTTP %d: %s", e.Status, strings.TrimSpace(e.Body))
}

// FetchAgentCard retrieves the agent's public AgentCard for discovery and
// enrichment.
func (c *Client) FetchAgentCard(ctx context.Context, baseURL string) (map[string]any, error) {
	return c.getJSON(ctx, baseURL+"/.well-known/agent-card.json")
}

// HealthCheck issues a GET against the given health path and reports whether
// the agent responded with a 2xx status.
func (c *Client) HealthCheck(ctx context.Context, baseURL, path string) bool {
	if path == "" {
		path = "/.well-known/agent-card.json"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (c *Client) getJSON(ctx context.Context, u string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode == http.StatusNotFound {
		return nil, &ErrNotFound{Path: u}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &ErrAgent{Status: resp.StatusCode, Body: string(data)}
	}
	var out map[string]any
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("a2a: decode response from %s: %w", u, err)
	}
	return out, nil
}
