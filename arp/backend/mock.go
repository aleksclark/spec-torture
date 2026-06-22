package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// MockOptions tunes the behaviour of an in-process mock agent.
type MockOptions struct {
	// DisableMessageSend makes POST /message:send return 404, mimicking an
	// agent that does not implement the messaging endpoint.
	DisableMessageSend bool
	// AlwaysMessage makes message:send return a bare Message instead of a Task.
	AlwaysMessage bool
}

// MockBackend runs agents as in-process A2A v1.0 HTTP+JSON servers. It is fully
// deterministic and requires no external processes, making it ideal for
// conformance testing and demo seeding.
type MockBackend struct {
	mu         sync.Mutex
	optsByTmpl map[string]MockOptions
	agents     []*mockAgent
}

// NewMockBackend returns an empty MockBackend.
func NewMockBackend() *MockBackend {
	return &MockBackend{optsByTmpl: map[string]MockOptions{}}
}

// SetTemplateOptions configures per-template mock behaviour.
func (b *MockBackend) SetTemplateOptions(template string, opts MockOptions) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.optsByTmpl[template] = opts
}

// Close shuts down all running mock agents.
func (b *MockBackend) Close() {
	b.mu.Lock()
	agents := b.agents
	b.agents = nil
	b.mu.Unlock()
	for _, a := range agents {
		_ = a.Stop(context.Background(), 0)
	}
}

// Spawn starts an in-process agent bound to the allocated port.
func (b *MockBackend) Spawn(_ context.Context, spec SpawnSpec) (Handle, error) {
	b.mu.Lock()
	opts := b.optsByTmpl[spec.Template.GetName()]
	b.mu.Unlock()

	addr := "127.0.0.1:" + strconv.Itoa(spec.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// Fall back to an ephemeral port if the requested one is unavailable.
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, fmt.Errorf("mock agent listen: %w", err)
		}
	}
	port := ln.Addr().(*net.TCPAddr).Port
	a := &mockAgent{
		port:      port,
		directURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		card:      mockCard(spec.Template, fmt.Sprintf("http://127.0.0.1:%d", port)),
		opts:      opts,
		tasks:     map[string]map[string]any{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.route)
	a.srv = &http.Server{Handler: mux}
	go func() { _ = a.srv.Serve(ln) }()

	b.mu.Lock()
	b.agents = append(b.agents, a)
	b.mu.Unlock()
	return a, nil
}

type mockAgent struct {
	port      int
	directURL string
	card      map[string]any
	opts      MockOptions
	srv       *http.Server

	mu      sync.Mutex
	taskSeq int
	msgSeq  int
	tasks   map[string]map[string]any
}

func (a *mockAgent) DirectURL() string { return a.directURL }
func (a *mockAgent) Port() int         { return a.port }
func (a *mockAgent) PID() int          { return 0 }

func (a *mockAgent) Stop(ctx context.Context, graceMs int) error {
	if a.srv == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	c, cancel := context.WithTimeout(ctx, time.Duration(maxInt(graceMs, 200))*time.Millisecond)
	defer cancel()
	return a.srv.Shutdown(c)
}

func (a *mockAgent) route(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/.well-known/agent-card.json":
		writeJSON(w, http.StatusOK, a.card)
	case r.URL.Path == "/message:send" && r.Method == http.MethodPost:
		a.handleSend(w, r)
	case strings.HasSuffix(r.URL.Path, ":cancel") && r.Method == http.MethodPost:
		a.handleCancel(w, r)
	case strings.HasPrefix(r.URL.Path, "/tasks/") && r.Method == http.MethodGet:
		a.handleGetTask(w, r)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (a *mockAgent) handleSend(w http.ResponseWriter, r *http.Request) {
	if a.opts.DisableMessageSend {
		http.Error(w, "message:send not implemented", http.StatusNotFound)
		return
	}
	var req struct {
		Message map[string]any `json:"message"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	text := extractText(req.Message)
	ctxID, _ := req.Message["context_id"].(string)

	a.mu.Lock()
	if ctxID == "" {
		ctxID = fmt.Sprintf("ctx-%d", a.taskSeq+1)
	}
	if a.opts.AlwaysMessage {
		a.msgSeq++
		msg := map[string]any{
			"message_id": fmt.Sprintf("msg-%d", a.msgSeq),
			"role":       "ROLE_AGENT",
			"parts":      []any{map[string]any{"text_part": map[string]any{"text": "echo: " + text}}},
			"context_id": ctxID,
		}
		a.mu.Unlock()
		writeJSON(w, http.StatusOK, msg)
		return
	}
	a.taskSeq++
	taskID := fmt.Sprintf("task-%d", a.taskSeq)
	task := a.newTask(taskID, ctxID, text)
	a.tasks[taskID] = task
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, task)
}

func (a *mockAgent) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/tasks/")
	a.mu.Lock()
	task, ok := a.tasks[id]
	a.mu.Unlock()
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	out := task
	if hl := r.URL.Query().Get("history_length"); hl != "" {
		if n, err := strconv.Atoi(hl); err == nil && n >= 0 {
			out = capHistory(task, n)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *mockAgent) handleCancel(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, ":cancel"), "/tasks/")
	a.mu.Lock()
	task, ok := a.tasks[id]
	if ok {
		task["status"] = map[string]any{"state": "TASK_STATE_CANCELED", "timestamp": now()}
		a.tasks[id] = task
	}
	a.mu.Unlock()
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (a *mockAgent) newTask(id, ctxID, text string) map[string]any {
	return map[string]any{
		"id":         id,
		"context_id": ctxID,
		"status":     map[string]any{"state": "TASK_STATE_COMPLETED", "timestamp": now()},
		"history": []any{
			map[string]any{"role": "ROLE_USER", "parts": []any{map[string]any{"text_part": map[string]any{"text": text}}}},
			map[string]any{"role": "ROLE_AGENT", "parts": []any{map[string]any{"text_part": map[string]any{"text": "echo: " + text}}}},
		},
		"artifacts": []any{
			map[string]any{"artifact_id": "art-1", "name": "echo", "parts": []any{map[string]any{"text_part": map[string]any{"text": "echo: " + text}}}},
		},
	}
}

func mockCard(t *arpv1.AgentTemplate, directURL string) map[string]any {
	name := t.GetName()
	desc := "Mock A2A agent"
	streaming := false
	var skills []any
	var inModes, outModes []string
	if cfg := t.GetA2ACardConfig(); cfg != nil {
		if cfg.GetName() != "" {
			name = cfg.GetName()
		}
		if cfg.GetDescription() != "" {
			desc = cfg.GetDescription()
		}
		streaming = cfg.GetStreaming()
		inModes = cfg.GetInputModes()
		outModes = cfg.GetOutputModes()
		for _, sk := range cfg.GetSkills() {
			tags := make([]any, 0, len(sk.GetTags()))
			for _, tg := range sk.GetTags() {
				tags = append(tags, tg)
			}
			skills = append(skills, map[string]any{
				"id":          sk.GetId(),
				"name":        sk.GetName(),
				"description": sk.GetDescription(),
				"tags":        tags,
			})
		}
	}
	if len(skills) == 0 {
		tags := make([]any, 0, len(t.GetCapabilities()))
		for _, c := range t.GetCapabilities() {
			tags = append(tags, c)
		}
		skills = []any{map[string]any{
			"id":          t.GetName(),
			"name":        t.GetName(),
			"description": desc,
			"tags":        tags,
		}}
	}
	if len(inModes) == 0 {
		inModes = []string{"text/plain"}
	}
	if len(outModes) == 0 {
		outModes = []string{"text/plain"}
	}
	return map[string]any{
		"name":        name,
		"description": desc,
		"version":     "1.0.0",
		"supported_interfaces": []any{
			map[string]any{"url": directURL, "transport": "HTTP_JSON"},
		},
		"capabilities":         map[string]any{"streaming": streaming},
		"skills":               skills,
		"default_input_modes":  toAnySlice(inModes),
		"default_output_modes": toAnySlice(outModes),
	}
}

func capHistory(task map[string]any, n int) map[string]any {
	out := make(map[string]any, len(task))
	for k, v := range task {
		out[k] = v
	}
	if hist, ok := task["history"].([]any); ok && len(hist) > n {
		out["history"] = hist[len(hist)-n:]
	}
	return out
}

func extractText(msg map[string]any) string {
	parts, _ := msg["parts"].([]any)
	var sb strings.Builder
	for _, p := range parts {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if tp, ok := pm["text_part"].(map[string]any); ok {
			if s, ok := tp["text"].(string); ok {
				sb.WriteString(s)
			}
		}
	}
	return sb.String()
}

func toAnySlice(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
