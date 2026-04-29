package clientrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aleksclark/spec-torture/internal/mockserver"
)

// A2AClientTests returns the full suite of A2A v1.0 client conformance tests.
func A2AClientTests() []TestCase {
	return []TestCase{
		// ── Discovery ────────────────────────────────────────────────────
		{
			ID:          "client-discovery-fetches-agent-card",
			Name:        "Client Fetches Agent Card",
			Description: "Client fetches /.well-known/agent-card.json via HTTP GET to discover agent capabilities.",
			Severity:    "required",
			Category:    "discovery",
			Tags:        []string{"discovery", "agent-card"},
			Timeout:     30 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.Discovery(ctx, mock.URL())
				if err != nil {
					return fmt.Errorf("discovery failed: %w", err)
				}

				reqs := mock.RequestsByMethod("")
				var found bool
				for _, r := range mock.Requests() {
					if r.Path == "/.well-known/agent-card.json" && r.HTTPMethod == "GET" {
						found = true
						break
					}
				}
				_ = reqs
				if !found {
					return fmt.Errorf("client did not GET /.well-known/agent-card.json")
				}
				return nil
			},
		},

		// ── SendMessage ──────────────────────────────────────────────────
		{
			ID:          "client-send-message-jsonrpc-envelope",
			Name:        "SendMessage Uses JSON-RPC 2.0 Envelope",
			Description: "Client's SendMessage request must use a valid JSON-RPC 2.0 envelope with jsonrpc, id, method, and params.",
			Severity:    "required",
			Category:    "messaging",
			Tags:        []string{"messaging", "jsonrpc"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Hello from test",
				})
				if err != nil {
					return fmt.Errorf("send message failed: %w", err)
				}

				reqs := mock.RequestsByMethod("SendMessage")
				if len(reqs) == 0 {
					return fmt.Errorf("no SendMessage request recorded")
				}

				return RequireJSONRPCEnvelope(reqs[0])
			},
		},
		{
			ID:          "client-send-message-method-name",
			Name:        "SendMessage Uses Correct Method Name",
			Description: "Client must use method name \"SendMessage\" (PascalCase, per v1.0 proto service definition).",
			Severity:    "required",
			Category:    "messaging",
			Tags:        []string{"messaging", "jsonrpc"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Hello from test",
				})
				if err != nil {
					return fmt.Errorf("send message failed: %w", err)
				}

				reqs := mock.RequestsByMethod("SendMessage")
				if len(reqs) == 0 {
					// Check if client used the old method name
					oldReqs := mock.RequestsByMethod("message/send")
					if len(oldReqs) > 0 {
						return fmt.Errorf("client used deprecated method \"message/send\" instead of v1.0 \"SendMessage\"")
					}
					return fmt.Errorf("no SendMessage request recorded")
				}
				return nil
			},
		},
		{
			ID:          "client-send-message-content-type",
			Name:        "SendMessage Uses application/json Content-Type",
			Description: "Client must send Content-Type: application/json header with JSON-RPC requests.",
			Severity:    "required",
			Category:    "messaging",
			Tags:        []string{"messaging", "headers"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Hello from test",
				})
				if err != nil {
					return fmt.Errorf("send message failed: %w", err)
				}

				reqs := mock.RequestsByMethod("SendMessage")
				if len(reqs) == 0 {
					return fmt.Errorf("no SendMessage request recorded")
				}

				return RequireContentType(reqs[0], "application/json")
			},
		},
		{
			ID:          "client-send-message-has-message-id",
			Name:        "SendMessage Includes messageId",
			Description: "Client must include a non-empty messageId in the message (REQUIRED per proto).",
			Severity:    "required",
			Category:    "messaging",
			Tags:        []string{"messaging", "fields"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Hello from test",
				})
				if err != nil {
					return fmt.Errorf("send message failed: %w", err)
				}

				reqs := mock.RequestsByMethod("SendMessage")
				if len(reqs) == 0 {
					return fmt.Errorf("no SendMessage request recorded")
				}

				var parsed map[string]any
				if err := json.Unmarshal([]byte(reqs[0].RawBody), &parsed); err != nil {
					return fmt.Errorf("failed to parse request: %w", err)
				}

				params, _ := parsed["params"].(map[string]any)
				if params == nil {
					return fmt.Errorf("request missing params")
				}
				msg, _ := params["message"].(map[string]any)
				if msg == nil {
					return fmt.Errorf("request missing params.message")
				}
				msgID, _ := msg["messageId"].(string)
				if msgID == "" {
					return fmt.Errorf("params.message.messageId is empty or missing (REQUIRED per proto)")
				}
				return nil
			},
		},
		{
			ID:          "client-send-message-has-role",
			Name:        "SendMessage Includes Role",
			Description: "Client must include role in the message (REQUIRED per proto). Expected value is \"user\" or \"ROLE_USER\".",
			Severity:    "required",
			Category:    "messaging",
			Tags:        []string{"messaging", "fields"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Hello from test",
				})
				if err != nil {
					return fmt.Errorf("send message failed: %w", err)
				}

				reqs := mock.RequestsByMethod("SendMessage")
				if len(reqs) == 0 {
					return fmt.Errorf("no SendMessage request recorded")
				}

				var parsed map[string]any
				json.Unmarshal([]byte(reqs[0].RawBody), &parsed)
				params, _ := parsed["params"].(map[string]any)
				msg, _ := params["message"].(map[string]any)
				if msg == nil {
					return fmt.Errorf("request missing params.message")
				}
				role, _ := msg["role"].(string)
				if role == "" {
					return fmt.Errorf("params.message.role is empty or missing (REQUIRED per proto)")
				}
				validRoles := map[string]bool{
					"user": true, "ROLE_USER": true,
					"agent": true, "ROLE_AGENT": true,
				}
				if !validRoles[role] {
					return fmt.Errorf("params.message.role %q is not a valid Role enum value", role)
				}
				return nil
			},
		},
		{
			ID:          "client-send-message-has-parts",
			Name:        "SendMessage Includes Parts",
			Description: "Client must include at least one part in the message (REQUIRED per proto).",
			Severity:    "required",
			Category:    "messaging",
			Tags:        []string{"messaging", "fields"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Hello from test",
				})
				if err != nil {
					return fmt.Errorf("send message failed: %w", err)
				}

				reqs := mock.RequestsByMethod("SendMessage")
				if len(reqs) == 0 {
					return fmt.Errorf("no SendMessage request recorded")
				}

				var parsed map[string]any
				json.Unmarshal([]byte(reqs[0].RawBody), &parsed)
				params, _ := parsed["params"].(map[string]any)
				msg, _ := params["message"].(map[string]any)
				if msg == nil {
					return fmt.Errorf("request missing params.message")
				}
				parts, _ := msg["parts"].([]any)
				if len(parts) == 0 {
					return fmt.Errorf("params.message.parts is empty or missing (REQUIRED per proto)")
				}
				return nil
			},
		},
		{
			ID:          "client-send-message-part-has-content",
			Name:        "SendMessage Part Has Content Field",
			Description: "Each Part must have exactly one content field set: text, raw, url, or data (oneof in proto).",
			Severity:    "required",
			Category:    "messaging",
			Tags:        []string{"messaging", "parts"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Hello from test",
				})
				if err != nil {
					return fmt.Errorf("send message failed: %w", err)
				}

				reqs := mock.RequestsByMethod("SendMessage")
				if len(reqs) == 0 {
					return fmt.Errorf("no SendMessage request recorded")
				}

				var parsed map[string]any
				json.Unmarshal([]byte(reqs[0].RawBody), &parsed)
				params, _ := parsed["params"].(map[string]any)
				msg, _ := params["message"].(map[string]any)
				parts, _ := msg["parts"].([]any)
				if len(parts) == 0 {
					return fmt.Errorf("no parts to check")
				}

				for i, p := range parts {
					part, ok := p.(map[string]any)
					if !ok {
						return fmt.Errorf("part[%d] is not an object", i)
					}
					hasContent := false
					for _, field := range []string{"text", "raw", "url", "data"} {
						if _, ok := part[field]; ok {
							hasContent = true
							break
						}
					}
					if !hasContent {
						return fmt.Errorf("part[%d] has no content field (text/raw/url/data)", i)
					}
				}
				return nil
			},
		},
		{
			ID:          "client-send-message-http-post",
			Name:        "SendMessage Uses HTTP POST",
			Description: "JSON-RPC requests must be sent via HTTP POST.",
			Severity:    "required",
			Category:    "messaging",
			Tags:        []string{"messaging", "transport"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Hello from test",
				})
				if err != nil {
					return fmt.Errorf("send message failed: %w", err)
				}

				reqs := mock.RequestsByMethod("SendMessage")
				if len(reqs) == 0 {
					return fmt.Errorf("no SendMessage request recorded")
				}

				if reqs[0].HTTPMethod != "POST" {
					return fmt.Errorf("expected HTTP POST, got %s", reqs[0].HTTPMethod)
				}
				return nil
			},
		},

		// ── GetTask ──────────────────────────────────────────────────────
		{
			ID:          "client-get-task-jsonrpc-envelope",
			Name:        "GetTask Uses JSON-RPC 2.0 Envelope",
			Description: "Client's GetTask request must use a valid JSON-RPC 2.0 envelope.",
			Severity:    "required",
			Category:    "lifecycle",
			Tags:        []string{"lifecycle", "task"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.GetTask(ctx, mock.URL(), "task-mock-001")
				if err != nil {
					return fmt.Errorf("get task failed: %w", err)
				}

				reqs := mock.RequestsByMethod("GetTask")
				if len(reqs) == 0 {
					return fmt.Errorf("no GetTask request recorded")
				}

				return RequireJSONRPCEnvelope(reqs[0])
			},
		},
		{
			ID:          "client-get-task-method-name",
			Name:        "GetTask Uses Correct Method Name",
			Description: "Client must use method name \"GetTask\" (PascalCase, per v1.0 proto).",
			Severity:    "required",
			Category:    "lifecycle",
			Tags:        []string{"lifecycle", "task"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.GetTask(ctx, mock.URL(), "task-mock-001")
				if err != nil {
					return fmt.Errorf("get task failed: %w", err)
				}

				reqs := mock.RequestsByMethod("GetTask")
				if len(reqs) == 0 {
					oldReqs := mock.RequestsByMethod("tasks/get")
					if len(oldReqs) > 0 {
						return fmt.Errorf("client used deprecated method \"tasks/get\" instead of v1.0 \"GetTask\"")
					}
					return fmt.Errorf("no GetTask request recorded")
				}
				return nil
			},
		},
		{
			ID:          "client-get-task-has-id",
			Name:        "GetTask Includes Task ID",
			Description: "Client must include the task ID in params (REQUIRED per proto).",
			Severity:    "required",
			Category:    "lifecycle",
			Tags:        []string{"lifecycle", "task"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.GetTask(ctx, mock.URL(), "task-mock-001")
				if err != nil {
					return fmt.Errorf("get task failed: %w", err)
				}

				reqs := mock.RequestsByMethod("GetTask")
				if len(reqs) == 0 {
					return fmt.Errorf("no GetTask request recorded")
				}

				return RequireRequestField(reqs[0], "params.id", "task-mock-001")
			},
		},

		// ── Context propagation ──────────────────────────────────────────
		{
			ID:          "client-send-message-context-propagation",
			Name:        "SendMessage Propagates contextId",
			Description: "When the server returns a contextId, subsequent messages should include it to maintain conversational context.",
			Severity:    "recommended",
			Category:    "messaging",
			Tags:        []string{"messaging", "context"},
			Timeout:     120 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				// First message — server returns contextId "ctx-mock-001"
				_, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "First message",
				})
				if err != nil {
					return fmt.Errorf("first message failed: %w", err)
				}

				// Second message — client should propagate the contextId
				_, err = driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Second message in same context",
				})
				if err != nil {
					return fmt.Errorf("second message failed: %w", err)
				}

				reqs := mock.RequestsByMethod("SendMessage")
				if len(reqs) < 2 {
					return fmt.Errorf("expected at least 2 SendMessage requests, got %d", len(reqs))
				}

				// Check second request for contextId
				var parsed map[string]any
				json.Unmarshal([]byte(reqs[1].RawBody), &parsed)
				params, _ := parsed["params"].(map[string]any)
				msg, _ := params["message"].(map[string]any)
				if msg == nil {
					return fmt.Errorf("second request missing params.message")
				}
				ctxID, _ := msg["contextId"].(string)
				if ctxID == "" {
					return fmt.Errorf("second message missing contextId (should propagate from first response)")
				}
				return nil
			},
		},

		// ── Response handling ────────────────────────────────────────────
		{
			ID:          "client-handles-task-response",
			Name:        "Client Handles Task Response",
			Description: "Client can handle a SendMessage response that contains a Task (one branch of the oneof).",
			Severity:    "required",
			Category:    "response-handling",
			Tags:        []string{"response-handling", "task"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				// Default mock handler returns a Task
				result, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Return a task",
				})
				if err != nil {
					return fmt.Errorf("send message returned error: %w", err)
				}
				if result == nil {
					return fmt.Errorf("send message returned nil result")
				}
				return nil
			},
		},
		{
			ID:          "client-handles-message-response",
			Name:        "Client Handles Message Response",
			Description: "Client can handle a SendMessage response that contains a Message (the other branch of the oneof).",
			Severity:    "required",
			Category:    "response-handling",
			Tags:        []string{"response-handling", "message"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				// Override handler to return a Message instead of a Task
				mock.SetHandler("SendMessage", func(req mockserver.Request) (any, *mockserver.JSONRPCError) {
					return map[string]any{
						"message": map[string]any{
							"messageId": "msg-resp-001",
							"role":      "agent",
							"parts": []any{
								map[string]any{"text": "Direct message response"},
							},
						},
					}, nil
				})

				result, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Return a message",
				})
				if err != nil {
					return fmt.Errorf("send message returned error when server responded with Message: %w", err)
				}
				if result == nil {
					return fmt.Errorf("send message returned nil result for Message response")
				}
				return nil
			},
		},
		{
			ID:          "client-handles-error-response",
			Name:        "Client Handles JSON-RPC Error",
			Description: "Client gracefully handles a JSON-RPC error response without crashing.",
			Severity:    "required",
			Category:    "response-handling",
			Tags:        []string{"response-handling", "errors"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				mock.SetHandler("SendMessage", func(req mockserver.Request) (any, *mockserver.JSONRPCError) {
					return nil, &mockserver.JSONRPCError{
						Code:    -32602,
						Message: "invalid params",
						Data:    map[string]any{"error": "intentional test error"},
					}
				})

				// The client should not crash — it should return an error
				_, err := driver.SendMessage(ctx, mock.URL(), map[string]any{
					"text": "Trigger an error",
				})
				// It's OK if the driver returns an error; what matters is it doesn't panic/hang
				_ = err
				return nil
			},
		},
		{
			ID:          "client-handles-task-not-found",
			Name:        "Client Handles Task Not Found",
			Description: "Client gracefully handles a task-not-found error from GetTask.",
			Severity:    "required",
			Category:    "response-handling",
			Tags:        []string{"response-handling", "errors"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				// Default handler returns -32001 for unknown task IDs
				_, err := driver.GetTask(ctx, mock.URL(), "nonexistent-task-id")
				// Should not crash; error return is acceptable
				_ = err
				return nil
			},
		},

		// ── Streaming ────────────────────────────────────────────────────
		{
			ID:          "client-streaming-method-name",
			Name:        "SendStreamingMessage Uses Correct Method Name",
			Description: "Client must use method name \"SendStreamingMessage\" (per v1.0 proto).",
			Severity:    "optional",
			Category:    "streaming",
			Tags:        []string{"streaming"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.SendStreamingMessage(ctx, mock.URL(), map[string]any{
					"text": "Stream this",
				})
				if err != nil {
					return fmt.Errorf("streaming message failed: %w", err)
				}

				reqs := mock.RequestsByMethod("SendStreamingMessage")
				if len(reqs) == 0 {
					oldReqs := mock.RequestsByMethod("message/stream")
					if len(oldReqs) > 0 {
						return fmt.Errorf("client used deprecated method \"message/stream\" instead of v1.0 \"SendStreamingMessage\"")
					}
					return fmt.Errorf("no SendStreamingMessage request recorded")
				}
				return nil
			},
		},
		{
			ID:          "client-streaming-accept-header",
			Name:        "Streaming Request Includes Accept: text/event-stream",
			Description: "Client should set Accept: text/event-stream header for streaming requests.",
			Severity:    "recommended",
			Category:    "streaming",
			Tags:        []string{"streaming", "headers"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, err := driver.SendStreamingMessage(ctx, mock.URL(), map[string]any{
					"text": "Stream this",
				})
				if err != nil {
					return fmt.Errorf("streaming message failed: %w", err)
				}

				reqs := mock.RequestsByMethod("SendStreamingMessage")
				if len(reqs) == 0 {
					return fmt.Errorf("no SendStreamingMessage request recorded")
				}

				accept := reqs[0].Headers.Get("Accept")
				if !strings.Contains(accept, "text/event-stream") {
					return fmt.Errorf("expected Accept header containing \"text/event-stream\", got %q", accept)
				}
				return nil
			},
		},

		// ── Protocol version ─────────────────────────────────────────────
		{
			ID:          "client-uses-v1-method-names",
			Name:        "Client Uses v1.0 PascalCase Method Names",
			Description: "All JSON-RPC method names should use PascalCase per the v1.0 proto service definition (SendMessage, not message/send).",
			Severity:    "required",
			Category:    "protocol",
			Tags:        []string{"protocol", "v1"},
			Timeout:     60 * time.Second,
			Run: func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error {
				_, _ = driver.SendMessage(ctx, mock.URL(), map[string]any{"text": "test"})
				_, _ = driver.GetTask(ctx, mock.URL(), "task-mock-001")

				deprecated := map[string]string{
					"message/send":    "SendMessage",
					"tasks/get":       "GetTask",
					"tasks/cancel":    "CancelTask",
					"message/stream":  "SendStreamingMessage",
					"tasks/resubscribe": "SubscribeToTask",
				}

				for _, req := range mock.Requests() {
					if req.Method == "" {
						continue
					}
					if correct, isOld := deprecated[req.Method]; isOld {
						return fmt.Errorf("client used deprecated method %q instead of v1.0 %q", req.Method, correct)
					}
				}
				return nil
			},
		},
	}
}
