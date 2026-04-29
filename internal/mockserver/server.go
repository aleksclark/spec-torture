package mockserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Request is a recorded JSON-RPC request from the client.
type Request struct {
	Timestamp time.Time      `json:"timestamp"`
	Method    string         `json:"method"`
	ID        any            `json:"id"`
	Params    map[string]any `json:"params,omitempty"`
	RawBody   string         `json:"raw_body"`
	Headers   http.Header    `json:"headers"`
	Path      string         `json:"path"`
	HTTPMethod string        `json:"http_method"`
}

// AgentCard is the mock agent card returned by the server.
type AgentCard struct {
	Name                string        `json:"name"`
	Description         string        `json:"description"`
	Version             string        `json:"version"`
	SupportedInterfaces []AgentIface  `json:"supportedInterfaces"`
	Capabilities        Capabilities  `json:"capabilities"`
	DefaultInputModes   []string      `json:"defaultInputModes"`
	DefaultOutputModes  []string      `json:"defaultOutputModes"`
	Skills              []Skill       `json:"skills"`
}

type AgentIface struct {
	URL             string `json:"url"`
	ProtocolBinding string `json:"protocolBinding"`
	ProtocolVersion string `json:"protocolVersion"`
}

type Capabilities struct {
	Streaming         bool `json:"streaming,omitempty"`
	PushNotifications bool `json:"pushNotifications,omitempty"`
}

type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// Server is a mock A2A server for testing clients.
type Server struct {
	mu       sync.Mutex
	requests []Request
	listener net.Listener
	server   *http.Server
	port     int
	card     AgentCard

	// Handler overrides — keyed by JSON-RPC method name.
	// If set, the handler function returns the JSON-RPC result (or error) to send.
	handlers map[string]func(req Request) (any, *JSONRPCError)

	// StreamHandler override — keyed by JSON-RPC method name.
	// Returns a sequence of SSE data lines.
	streamHandlers map[string]func(req Request) []string
}

// JSONRPCError is a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// New creates a mock A2A server on a random available port.
func New() (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	s := &Server{
		listener: ln,
		port:     port,
		card: AgentCard{
			Name:        "mock-a2a-agent",
			Description: "Mock A2A agent for client conformance testing",
			Version:     "1.0.0",
			SupportedInterfaces: []AgentIface{
				{
					URL:             fmt.Sprintf("http://127.0.0.1:%d/", port),
					ProtocolBinding: "JSONRPC",
					ProtocolVersion: "1.0",
				},
			},
			Capabilities: Capabilities{
				Streaming: true,
			},
			DefaultInputModes:  []string{"text/plain"},
			DefaultOutputModes: []string{"text/plain"},
			Skills: []Skill{
				{
					ID:          "echo",
					Name:        "Echo",
					Description: "Echoes back whatever you send",
					Tags:        []string{"test"},
				},
			},
		},
		handlers:       make(map[string]func(req Request) (any, *JSONRPCError)),
		streamHandlers: make(map[string]func(req Request) []string),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/agent-card.json", s.handleAgentCard)
	mux.HandleFunc("/_inspect/requests", s.handleInspectRequests)
	mux.HandleFunc("/_inspect/reset", s.handleInspectReset)
	// ARP REST endpoints
	mux.HandleFunc("/a2a/agents", s.handleARPAgents)
	mux.HandleFunc("/a2a/discover", s.handleARPDiscover)
	mux.HandleFunc("/a2a/route/", s.handleARPRoute)
	mux.HandleFunc("/api/workspaces", s.handleARPWorkspaces)
	mux.HandleFunc("/api/projects", s.handleARPProjects)
	// Catch-all: JSON-RPC or ARP agent-specific paths
	mux.HandleFunc("/", s.handleCatchAll)

	s.server = &http.Server{Handler: mux}
	go s.server.Serve(ln)

	return s, nil
}

// URL returns the base URL of the mock server.
func (s *Server) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", s.port)
}

// Port returns the port the mock server is listening on.
func (s *Server) Port() int {
	return s.port
}

// Requests returns a copy of all recorded requests.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Request, len(s.requests))
	copy(out, s.requests)
	return out
}

// RequestsByMethod returns recorded requests filtered by JSON-RPC method.
func (s *Server) RequestsByMethod(method string) []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Request
	for _, r := range s.requests {
		if r.Method == method {
			out = append(out, r)
		}
	}
	return out
}

// Reset clears all recorded requests.
func (s *Server) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = nil
}

// Close shuts down the mock server.
func (s *Server) Close() error {
	return s.server.Close()
}

// SetHandler sets a custom handler for a specific JSON-RPC method.
func (s *Server) SetHandler(method string, h func(req Request) (any, *JSONRPCError)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = h
}

// SetStreamHandler sets a custom streaming handler for a specific JSON-RPC method.
func (s *Server) SetStreamHandler(method string, h func(req Request) []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streamHandlers[method] = h
}

func (s *Server) record(r Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, r)
}

func (s *Server) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	s.record(Request{
		Timestamp:  time.Now(),
		Path:       r.URL.Path,
		HTTPMethod: r.Method,
		Headers:    r.Header,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.card)
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONRPCError(w, nil, -32700, "parse error", nil)
		return
	}

	var rpcReq struct {
		JSONRPC string         `json:"jsonrpc"`
		ID      any            `json:"id"`
		Method  string         `json:"method"`
		Params  map[string]any `json:"params"`
	}
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		writeJSONRPCError(w, nil, -32700, "parse error", nil)
		return
	}

	if rpcReq.JSONRPC != "2.0" {
		writeJSONRPCError(w, rpcReq.ID, -32600, "invalid request", nil)
		return
	}

	req := Request{
		Timestamp:  time.Now(),
		Method:     rpcReq.Method,
		ID:         rpcReq.ID,
		Params:     rpcReq.Params,
		RawBody:    string(body),
		Headers:    r.Header,
		Path:       r.URL.Path,
		HTTPMethod: r.Method,
	}
	s.record(req)

	// Check for streaming handler first
	s.mu.Lock()
	streamH, hasStream := s.streamHandlers[rpcReq.Method]
	s.mu.Unlock()

	if hasStream && strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		events := streamH(req)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, _ := w.(http.Flusher)
		for _, e := range events {
			fmt.Fprintf(w, "data: %s\n\n", e)
			if flusher != nil {
				flusher.Flush()
			}
		}
		return
	}

	// Check for custom handler
	s.mu.Lock()
	h, ok := s.handlers[rpcReq.Method]
	s.mu.Unlock()

	if ok {
		result, rpcErr := h(req)
		if rpcErr != nil {
			writeJSONRPCError(w, rpcReq.ID, rpcErr.Code, rpcErr.Message, rpcErr.Data)
			return
		}
		writeJSONRPCResult(w, rpcReq.ID, result)
		return
	}

	// Default handlers
	switch rpcReq.Method {
	case "SendMessage":
		s.handleSendMessage(w, rpcReq.ID, rpcReq.Params)
	case "GetTask":
		s.handleGetTask(w, rpcReq.ID, rpcReq.Params)
	case "CancelTask":
		s.handleCancelTask(w, rpcReq.ID, rpcReq.Params)
	case "SendStreamingMessage":
		s.handleSendStreamingMessage(w, r, rpcReq.ID, rpcReq.Params)
	default:
		writeJSONRPCError(w, rpcReq.ID, -32601, "method not found", nil)
	}
}

func (s *Server) handleSendMessage(w http.ResponseWriter, id any, params map[string]any) {
	msg, _ := params["message"].(map[string]any)
	msgID, _ := msg["messageId"].(string)
	if msgID == "" {
		writeJSONRPCError(w, id, -32602, "invalid params", map[string]any{
			"error": "message ID is required",
		})
		return
	}

	contextID := "ctx-mock-001"
	if cid, ok := msg["contextId"].(string); ok && cid != "" {
		contextID = cid
	}

	// Return a Task wrapped in SendMessageResponse oneof
	result := map[string]any{
		"task": map[string]any{
			"id":        "task-mock-001",
			"contextId": contextID,
			"status": map[string]any{
				"state": "completed",
				"message": map[string]any{
					"messageId": "resp-mock-001",
					"role":      "agent",
					"parts": []any{
						map[string]any{"text": "Hello from mock agent"},
					},
				},
			},
		},
	}
	writeJSONRPCResult(w, id, result)
}

func (s *Server) handleGetTask(w http.ResponseWriter, id any, params map[string]any) {
	taskID, _ := params["id"].(string)
	if taskID == "" {
		writeJSONRPCError(w, id, -32602, "invalid params", nil)
		return
	}

	if taskID == "task-mock-001" {
		result := map[string]any{
			"id":        "task-mock-001",
			"contextId": "ctx-mock-001",
			"status": map[string]any{
				"state": "completed",
			},
		}
		writeJSONRPCResult(w, id, result)
		return
	}

	writeJSONRPCError(w, id, -32001, "task not found", nil)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, id any, params map[string]any) {
	taskID, _ := params["id"].(string)
	if taskID == "" {
		writeJSONRPCError(w, id, -32602, "invalid params", nil)
		return
	}

	result := map[string]any{
		"id":        taskID,
		"contextId": "ctx-mock-001",
		"status": map[string]any{
			"state": "canceled",
		},
	}
	writeJSONRPCResult(w, id, result)
}

func (s *Server) handleSendStreamingMessage(w http.ResponseWriter, r *http.Request, id any, params map[string]any) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONRPCError(w, id, -32603, "streaming not supported", nil)
		return
	}

	// Send a status update event
	statusEvent := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"statusUpdate": map[string]any{
				"taskId":    "task-stream-001",
				"contextId": "ctx-mock-001",
				"status": map[string]any{
					"state": "working",
				},
			},
		},
	}
	data, _ := json.Marshal(statusEvent)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	// Send a final task completion event
	taskEvent := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"task": map[string]any{
				"id":        "task-stream-001",
				"contextId": "ctx-mock-001",
				"status": map[string]any{
					"state": "completed",
					"message": map[string]any{
						"messageId": "resp-stream-001",
						"role":      "agent",
						"parts": []any{
							map[string]any{"text": "Streamed response from mock agent"},
						},
					},
				},
			},
		},
	}
	data, _ = json.Marshal(taskEvent)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func writeJSONRPCResult(w http.ResponseWriter, id any, result any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

func writeJSONRPCError(w http.ResponseWriter, id any, code int, message string, data any) {
	w.Header().Set("Content-Type", "application/json")
	errObj := map[string]any{
		"code":    code,
		"message": message,
	}
	if data != nil {
		errObj["data"] = data
	}
	json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   errObj,
	})
}

// handleInspectRequests returns all recorded requests as JSON.
func (s *Server) handleInspectRequests(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Query().Get("method")
	var reqs []Request
	if method != "" {
		reqs = s.RequestsByMethod(method)
	} else {
		reqs = s.Requests()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reqs)
}

// handleInspectReset clears all recorded requests.
func (s *Server) handleInspectReset(w http.ResponseWriter, r *http.Request) {
	s.Reset()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}
