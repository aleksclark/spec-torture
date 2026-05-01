package mockserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// arpState holds in-memory state for ARP lifecycle simulation.
type arpState struct {
	mu         sync.RWMutex
	projects   map[string]map[string]any   // name -> project
	workspaces map[string]map[string]any   // name -> workspace
	agents     map[string]map[string]any   // id -> agent instance
	nextPort   int
	nextAgentN int
}

func newARPState() *arpState {
	return &arpState{
		projects:   make(map[string]map[string]any),
		workspaces: make(map[string]map[string]any),
		agents:     make(map[string]map[string]any),
		nextPort:   10001,
		nextAgentN: 1,
	}
}

// registerARPLifecycleHandlers registers full CRUD handlers for ARP lifecycle testing.
func (s *Server) registerARPLifecycleHandlers(mux *http.ServeMux) {
	s.arp = newARPState()
	mux.HandleFunc("/api/projects", s.handleARPProjectsCRUD)
	mux.HandleFunc("/api/projects/", s.handleARPProjectByName)
	mux.HandleFunc("/api/workspaces", s.handleARPWorkspacesCRUD)
	mux.HandleFunc("/api/workspaces/", s.handleARPWorkspaceByName)
	mux.HandleFunc("/api/agents", s.handleARPAgentsCRUD)
	mux.HandleFunc("/api/agents/", s.handleARPAgentByID)
}

// handleARPProjectsCRUD handles GET /api/projects and POST /api/projects
func (s *Server) handleARPProjectsCRUD(w http.ResponseWriter, r *http.Request) {
	s.recordHTTP(r)
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		s.arp.mu.RLock()
		projects := make([]map[string]any, 0, len(s.arp.projects))
		for _, p := range s.arp.projects {
			projects = append(projects, p)
		}
		s.arp.mu.RUnlock()
		json.NewEncoder(w).Encode(projects)

	case "POST":
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeHTTPError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		name, _ := body["name"].(string)
		if name == "" {
			writeHTTPError(w, http.StatusBadRequest, "name is required")
			return
		}

		s.arp.mu.Lock()
		if _, exists := s.arp.projects[name]; exists {
			s.arp.mu.Unlock()
			writeHTTPError(w, http.StatusConflict, fmt.Sprintf("project %q already exists", name))
			return
		}

		project := map[string]any{
			"name":       name,
			"repo":       body["repo"],
			"created_at": time.Now().Format(time.RFC3339),
		}
		if agents, ok := body["agents"]; ok {
			project["agents"] = agents
		}
		s.arp.projects[name] = project
		s.arp.mu.Unlock()

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(project)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleARPProjectByName handles GET/DELETE /api/projects/{name}
func (s *Server) handleARPProjectByName(w http.ResponseWriter, r *http.Request) {
	s.recordHTTP(r)
	w.Header().Set("Content-Type", "application/json")

	name := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	if name == "" {
		writeHTTPError(w, http.StatusBadRequest, "project name required")
		return
	}

	switch r.Method {
	case "GET":
		s.arp.mu.RLock()
		project, exists := s.arp.projects[name]
		s.arp.mu.RUnlock()
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "project not found"})
			return
		}
		json.NewEncoder(w).Encode(project)

	case "DELETE":
		s.arp.mu.Lock()
		_, exists := s.arp.projects[name]
		if !exists {
			s.arp.mu.Unlock()
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "project not found"})
			return
		}

		// Check for active workspaces
		for _, ws := range s.arp.workspaces {
			if ws["project"] == name && ws["status"] == "active" {
				s.arp.mu.Unlock()
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]any{"error": "project has active workspaces"})
				return
			}
		}

		delete(s.arp.projects, name)
		s.arp.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "name": name})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleARPWorkspacesCRUD handles GET/POST /api/workspaces
func (s *Server) handleARPWorkspacesCRUD(w http.ResponseWriter, r *http.Request) {
	s.recordHTTP(r)
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		s.arp.mu.RLock()
		workspaces := make([]map[string]any, 0, len(s.arp.workspaces))
		project := r.URL.Query().Get("project")
		status := r.URL.Query().Get("status")
		for _, ws := range s.arp.workspaces {
			if project != "" && ws["project"] != project {
				continue
			}
			if status != "" && ws["status"] != status {
				continue
			}
			workspaces = append(workspaces, ws)
		}
		s.arp.mu.RUnlock()
		json.NewEncoder(w).Encode(workspaces)

	case "POST":
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeHTTPError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		name, _ := body["name"].(string)
		project, _ := body["project"].(string)
		if name == "" || project == "" {
			writeHTTPError(w, http.StatusBadRequest, "name and project are required")
			return
		}

		s.arp.mu.Lock()
		if _, exists := s.arp.projects[project]; !exists {
			s.arp.mu.Unlock()
			writeHTTPError(w, http.StatusBadRequest, fmt.Sprintf("project %q does not exist", project))
			return
		}

		ws := map[string]any{
			"name":       name,
			"project":    project,
			"status":     "active",
			"agents":     []any{},
			"dir":        fmt.Sprintf("/tmp/workspaces/%s", name),
			"created_at": time.Now().Format(time.RFC3339),
		}
		s.arp.workspaces[name] = ws

		// Handle auto_agents
		if autoAgents, ok := body["auto_agents"].([]any); ok && len(autoAgents) > 0 {
			projectData := s.arp.projects[project]
			templates, _ := projectData["agents"].([]any)
			agentList := []any{}
			for _, tmplName := range autoAgents {
				tmplStr, _ := tmplName.(string)
				for _, t := range templates {
					tmplMap, _ := t.(map[string]any)
					if tmplMap["name"] == tmplStr {
						agentID := fmt.Sprintf("agent-%03d", s.arp.nextAgentN)
						s.arp.nextAgentN++
						port := s.arp.nextPort
						s.arp.nextPort++
						agent := map[string]any{
							"id":         agentID,
							"template":   tmplStr,
							"workspace":  name,
							"status":     "starting",
							"port":       port,
							"direct_url": fmt.Sprintf("http://127.0.0.1:%d", port),
							"proxy_url":  fmt.Sprintf("%s/a2a/agents/%s/", s.URL(), agentID),
							"started_at": time.Now().Format(time.RFC3339),
						}
						s.arp.agents[agentID] = agent
						agentList = append(agentList, agent)
						// Mark as ready after "starting"
						go func(id string) {
							time.Sleep(100 * time.Millisecond)
							s.arp.mu.Lock()
							if a, ok := s.arp.agents[id]; ok {
								a["status"] = "ready"
							}
							s.arp.mu.Unlock()
						}(agentID)
					}
				}
			}
			ws["agents"] = agentList
		}

		s.arp.mu.Unlock()

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ws)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleARPWorkspaceByName handles GET/DELETE /api/workspaces/{name}
func (s *Server) handleARPWorkspaceByName(w http.ResponseWriter, r *http.Request) {
	s.recordHTTP(r)
	w.Header().Set("Content-Type", "application/json")

	name := strings.TrimPrefix(r.URL.Path, "/api/workspaces/")
	if name == "" {
		writeHTTPError(w, http.StatusBadRequest, "workspace name required")
		return
	}

	switch r.Method {
	case "GET":
		s.arp.mu.RLock()
		ws, exists := s.arp.workspaces[name]
		if !exists {
			s.arp.mu.RUnlock()
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "workspace not found"})
			return
		}
		// Include agents list
		agents := []any{}
		for _, a := range s.arp.agents {
			if a["workspace"] == name {
				agents = append(agents, a)
			}
		}
		result := make(map[string]any)
		for k, v := range ws {
			result[k] = v
		}
		result["agents"] = agents
		s.arp.mu.RUnlock()
		json.NewEncoder(w).Encode(result)

	case "DELETE":
		s.arp.mu.Lock()
		_, exists := s.arp.workspaces[name]
		if !exists {
			s.arp.mu.Unlock()
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "workspace not found"})
			return
		}

		// Stop all agents in this workspace
		for id, a := range s.arp.agents {
			if a["workspace"] == name {
				a["status"] = "stopped"
				delete(s.arp.agents, id)
			}
		}

		delete(s.arp.workspaces, name)
		s.arp.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "name": name})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleARPAgentsCRUD handles GET/POST /api/agents
func (s *Server) handleARPAgentsCRUD(w http.ResponseWriter, r *http.Request) {
	s.recordHTTP(r)
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		s.arp.mu.RLock()
		agents := make([]map[string]any, 0, len(s.arp.agents))
		wsFilter := r.URL.Query().Get("workspace")
		statusFilter := r.URL.Query().Get("status")
		tmplFilter := r.URL.Query().Get("template")
		for _, a := range s.arp.agents {
			if wsFilter != "" && a["workspace"] != wsFilter {
				continue
			}
			if statusFilter != "" && a["status"] != statusFilter {
				continue
			}
			if tmplFilter != "" && a["template"] != tmplFilter {
				continue
			}
			agents = append(agents, a)
		}
		s.arp.mu.RUnlock()
		json.NewEncoder(w).Encode(agents)

	case "POST":
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeHTTPError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		workspace, _ := body["workspace"].(string)
		template, _ := body["template"].(string)
		if workspace == "" || template == "" {
			writeHTTPError(w, http.StatusBadRequest, "workspace and template are required")
			return
		}

		s.arp.mu.Lock()
		if _, exists := s.arp.workspaces[workspace]; !exists {
			s.arp.mu.Unlock()
			writeHTTPError(w, http.StatusBadRequest, fmt.Sprintf("workspace %q does not exist", workspace))
			return
		}

		agentID := fmt.Sprintf("agent-%03d", s.arp.nextAgentN)
		s.arp.nextAgentN++
		port := s.arp.nextPort
		s.arp.nextPort++

		agent := map[string]any{
			"id":         agentID,
			"template":   template,
			"workspace":  workspace,
			"status":     "starting",
			"port":       port,
			"direct_url": fmt.Sprintf("http://127.0.0.1:%d", port),
			"proxy_url":  fmt.Sprintf("%s/a2a/agents/%s/", s.URL(), agentID),
			"started_at": time.Now().Format(time.RFC3339),
		}

		if name, ok := body["name"]; ok {
			agent["name"] = name
		}
		if tokenScope, ok := body["scope"]; ok {
			agent["token_id"] = fmt.Sprintf("tok-%s", agentID)
			agent["scope"] = tokenScope
		}
		if perm, ok := body["permission"]; ok {
			agent["token_id"] = fmt.Sprintf("tok-%s", agentID)
			agent["permission"] = perm
		}

		s.arp.agents[agentID] = agent
		s.arp.mu.Unlock()

		// Simulate agent startup: transition to ready after a short delay
		go func(id string) {
			time.Sleep(100 * time.Millisecond)
			s.arp.mu.Lock()
			if a, ok := s.arp.agents[id]; ok {
				a["status"] = "ready"
			}
			s.arp.mu.Unlock()
		}(agentID)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(agent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleARPAgentByID handles GET/POST /api/agents/{id}[/stop|/restart]
func (s *Server) handleARPAgentByID(w http.ResponseWriter, r *http.Request) {
	s.recordHTTP(r)
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	parts := strings.SplitN(path, "/", 2)
	agentID := parts[0]
	subAction := ""
	if len(parts) > 1 {
		subAction = parts[1]
	}

	switch {
	case r.Method == "GET" && subAction == "":
		// GET /api/agents/{id} - agent status
		s.arp.mu.RLock()
		agent, exists := s.arp.agents[agentID]
		s.arp.mu.RUnlock()
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "agent not found"})
			return
		}

		// Add mock a2a_agent_card
		result := make(map[string]any)
		for k, v := range agent {
			result[k] = v
		}
		result["a2a_agent_card"] = map[string]any{
			"name":    agent["template"],
			"version": "1.0.0",
			"metadata": map[string]any{
				"arp": map[string]any{
					"agent_id":  agentID,
					"workspace": agent["workspace"],
					"status":    agent["status"],
				},
			},
		}
		json.NewEncoder(w).Encode(result)

	case r.Method == "POST" && subAction == "stop":
		// POST /api/agents/{id}/stop
		s.arp.mu.Lock()
		agent, exists := s.arp.agents[agentID]
		if !exists {
			s.arp.mu.Unlock()
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "agent not found"})
			return
		}
		agent["status"] = "stopped"
		result := make(map[string]any)
		for k, v := range agent {
			result[k] = v
		}
		s.arp.mu.Unlock()
		json.NewEncoder(w).Encode(result)

	case r.Method == "POST" && subAction == "restart":
		// POST /api/agents/{id}/restart
		s.arp.mu.Lock()
		agent, exists := s.arp.agents[agentID]
		if !exists {
			s.arp.mu.Unlock()
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "agent not found"})
			return
		}
		agent["status"] = "starting"
		newPort := s.arp.nextPort
		s.arp.nextPort++
		agent["port"] = newPort
		agent["direct_url"] = fmt.Sprintf("http://127.0.0.1:%d", newPort)
		result := make(map[string]any)
		for k, v := range agent {
			result[k] = v
		}
		s.arp.mu.Unlock()

		// Transition to ready
		go func(id string) {
			time.Sleep(100 * time.Millisecond)
			s.arp.mu.Lock()
			if a, ok := s.arp.agents[id]; ok {
				a["status"] = "ready"
			}
			s.arp.mu.Unlock()
		}(agentID)

		json.NewEncoder(w).Encode(result)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) recordHTTP(r *http.Request) {
	s.record(Request{
		Timestamp:  time.Now(),
		Path:       r.URL.Path,
		HTTPMethod: r.Method,
		Headers:    r.Header,
	})
}

func writeHTTPError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"error": message})
}
