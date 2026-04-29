package mockserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// handleCatchAll routes to JSON-RPC or records as a generic REST request.
func (s *Server) handleCatchAll(w http.ResponseWriter, r *http.Request) {
	// ARP agent-specific paths like /a2a/agents/{id}/...
	if strings.HasPrefix(r.URL.Path, "/a2a/agents/") {
		s.handleARPAgentSubpath(w, r)
		return
	}
	// Otherwise assume JSON-RPC
	s.handleRPC(w, r)
}

// handleARPAgents handles GET /a2a/agents
func (s *Server) handleARPAgents(w http.ResponseWriter, r *http.Request) {
	// Only match exact /a2a/agents (sub-paths like /a2a/agents/foo go to catch-all)
	if r.URL.Path != "/a2a/agents" {
		s.handleARPAgentSubpath(w, r)
		return
	}

	s.record(Request{
		Timestamp:  time.Now(),
		Path:       r.URL.Path,
		HTTPMethod: r.Method,
		Headers:    r.Header,
	})

	agents := []map[string]any{
		{
			"name":    "mock-a2a-agent",
			"version": "1.0.0",
			"supportedInterfaces": []map[string]any{
				{"url": s.URL() + "/a2a/agents/mock-agent-001/", "protocolBinding": "JSONRPC", "protocolVersion": "1.0"},
			},
			"capabilities":     map[string]any{"streaming": true},
			"defaultInputModes":  []string{"text/plain"},
			"defaultOutputModes": []string{"text/plain"},
			"skills": []map[string]any{
				{"id": "echo", "name": "Echo", "description": "Echoes back", "tags": []string{"echo", "test"}},
			},
			"metadata": map[string]any{
				"arp": map[string]any{
					"agent_id":   "mock-agent-001",
					"workspace":  "test-workspace",
					"project":    "test-project",
					"template":   "default",
					"status":     "ready",
					"direct_url": "http://127.0.0.1:9999",
					"started_at": "2026-01-01T00:00:00Z",
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

// handleARPDiscover handles GET /a2a/discover
func (s *Server) handleARPDiscover(w http.ResponseWriter, r *http.Request) {
	s.record(Request{
		Timestamp:  time.Now(),
		Path:       r.URL.Path,
		HTTPMethod: r.Method,
		Headers:    r.Header,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"agents": []map[string]any{
			{"agent_id": "mock-agent-001", "name": "mock-a2a-agent", "status": "ready"},
		},
	})
}

// handleARPAgentSubpath handles /a2a/agents/{id}/...
func (s *Server) handleARPAgentSubpath(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	s.record(Request{
		Timestamp:  time.Now(),
		Path:       r.URL.Path,
		HTTPMethod: r.Method,
		Headers:    r.Header,
		RawBody:    string(body),
	})

	// Parse agent ID from path: /a2a/agents/{id}/...
	path := strings.TrimPrefix(r.URL.Path, "/a2a/agents/")
	parts := strings.SplitN(path, "/", 2)
	agentID := parts[0]

	// Check for nonexistent agents
	if agentID == "nonexistent-agent-00000" {
		http.Error(w, `{"error": "agent not found"}`, http.StatusNotFound)
		return
	}

	subpath := ""
	if len(parts) > 1 {
		subpath = parts[1]
	}

	w.Header().Set("Content-Type", "application/json")

	switch {
	case subpath == ".well-known/agent-card.json":
		json.NewEncoder(w).Encode(map[string]any{
			"name":    "mock-a2a-agent",
			"version": "1.0.0",
			"supportedInterfaces": []map[string]any{
				{"url": s.URL() + "/a2a/agents/" + agentID + "/", "protocolBinding": "JSONRPC", "protocolVersion": "1.0"},
			},
			"skills": []map[string]any{
				{"id": "echo", "name": "Echo", "description": "Echoes back", "tags": []string{"echo", "test"}},
			},
			"metadata": map[string]any{
				"arp": map[string]any{
					"agent_id":   agentID,
					"workspace":  "test-workspace",
					"project":    "test-project",
					"template":   "default",
					"status":     "ready",
					"direct_url": "http://127.0.0.1:9999",
					"started_at": "2026-01-01T00:00:00Z",
				},
			},
		})

	case strings.HasSuffix(subpath, "message:send"):
		json.NewEncoder(w).Encode(map[string]any{
			"task": map[string]any{
				"id":        "task-arp-001",
				"contextId": "ctx-arp-001",
				"status": map[string]any{
					"state": "completed",
					"message": map[string]any{
						"messageId": "resp-arp-001",
						"role":      "agent",
						"parts":     []any{map[string]any{"text": "ARP proxy response"}},
					},
				},
			},
		})

	case strings.HasSuffix(subpath, "message:stream"):
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		data, _ := json.Marshal(map[string]any{
			"task": map[string]any{"id": "task-stream-001", "status": map[string]any{"state": "completed"}},
		})
		w.Write([]byte("data: "))
		w.Write(data)
		w.Write([]byte("\n\n"))
		flusher.Flush()

	default:
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "agent_id": agentID, "subpath": subpath})
	}
}

// handleARPRoute handles POST /a2a/route/...
func (s *Server) handleARPRoute(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	s.record(Request{
		Timestamp:  time.Now(),
		Path:       r.URL.Path,
		HTTPMethod: r.Method,
		Headers:    r.Header,
		RawBody:    string(body),
	})

	w.Header().Set("Content-Type", "application/json")

	// Parse body to check routing tags
	var reqBody map[string]any
	json.Unmarshal(body, &reqBody)

	routing, _ := reqBody["routing"].(map[string]any)
	tags, _ := routing["tags"].([]any)

	// Return 404 for nonexistent capabilities
	for _, t := range tags {
		if ts, ok := t.(string); ok && strings.Contains(ts, "nonexistent") {
			http.Error(w, `{"error": "no matching agent"}`, http.StatusNotFound)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]any{
		"task": map[string]any{
			"id":        "task-routed-001",
			"contextId": "ctx-routed-001",
			"status":    map[string]any{"state": "completed"},
		},
	})
}

// handleARPWorkspaces handles GET /api/workspaces
func (s *Server) handleARPWorkspaces(w http.ResponseWriter, r *http.Request) {
	s.record(Request{
		Timestamp:  time.Now(),
		Path:       r.URL.Path,
		HTTPMethod: r.Method,
		Headers:    r.Header,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]any{
		{"name": "test-workspace", "project": "test-project", "active": true, "dir": "/tmp/test"},
	})
}

// handleARPProjects handles GET /api/projects
func (s *Server) handleARPProjects(w http.ResponseWriter, r *http.Request) {
	s.record(Request{
		Timestamp:  time.Now(),
		Path:       r.URL.Path,
		HTTPMethod: r.Method,
		Headers:    r.Header,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]any{
		{"name": "test-project", "repo": "https://github.com/test/test", "branch": "main"},
	})
}
