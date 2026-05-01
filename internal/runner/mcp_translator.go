package runner

import (
	"fmt"
	"strings"

	"github.com/aleksclark/spec-torture/internal/schema"
)

// mcpToolRoute defines how an MCP tool name maps to an HTTP REST endpoint.
type mcpToolRoute struct {
	Method string
	Path   string
	// BodyFromArgs: if true, the arguments map is sent as the request body.
	BodyFromArgs bool
	// PathParam: if set, this argument is extracted and appended to the path.
	PathParam string
	// QueryParams: arguments that become query parameters instead of body fields.
	QueryParams []string
}

// mcpToolRoutes maps MCP tool names to HTTP REST endpoints.
// These correspond to the ARP REST API endpoints.
var mcpToolRoutes = map[string]mcpToolRoute{
	// Project operations
	"project/register": {
		Method:       "POST",
		Path:         "/api/projects",
		BodyFromArgs: true,
	},
	"project/unregister": {
		Method:    "DELETE",
		Path:      "/api/projects",
		PathParam: "name",
	},
	"project/list": {
		Method: "GET",
		Path:   "/api/projects",
	},
	"project/get": {
		Method:    "GET",
		Path:      "/api/projects",
		PathParam: "name",
	},

	// Workspace operations
	"workspace/create": {
		Method:       "POST",
		Path:         "/api/workspaces",
		BodyFromArgs: true,
	},
	"workspace/destroy": {
		Method:    "DELETE",
		Path:      "/api/workspaces",
		PathParam: "name",
	},
	"workspace/list": {
		Method:      "GET",
		Path:        "/api/workspaces",
		QueryParams: []string{"project", "status"},
	},
	"workspace/get": {
		Method:    "GET",
		Path:      "/api/workspaces",
		PathParam: "name",
	},

	// Agent operations
	"agent/spawn": {
		Method:       "POST",
		Path:         "/api/agents",
		BodyFromArgs: true,
	},
	"agent/stop": {
		Method:    "POST",
		Path:      "/api/agents",
		PathParam: "agent_id",
		// The sub-action is appended as /stop
	},
	"agent/restart": {
		Method:    "POST",
		Path:      "/api/agents",
		PathParam: "agent_id",
	},
	"agent/list": {
		Method:      "GET",
		Path:        "/api/agents",
		QueryParams: []string{"workspace", "status", "template"},
	},
	"agent/status": {
		Method:    "GET",
		Path:      "/api/agents",
		PathParam: "agent_id",
	},

	// Messaging operations
	"agent/send_message": {
		Method:       "POST",
		Path:         "/a2a/agents",
		PathParam:    "agent_id",
		BodyFromArgs: true,
	},
}

// translateMCPToolCall converts an mcp_tool_call step into an HTTP send step.
// It maps MCP tool names to REST API endpoints using the mcpToolRoutes table.
func translateMCPToolCall(step *schema.Step, transport Transport) *schema.Step {
	payload := step.Payload
	if transport.LastResponse() != nil {
		payload = interpolateVars(payload, transport.LastResponse())
	}

	toolName, _ := payload["tool"].(string)
	args, _ := payload["arguments"].(map[string]any)
	if args == nil {
		args = make(map[string]any)
	}

	route, ok := mcpToolRoutes[toolName]
	if !ok {
		// Fallback: convert tool name to a REST path
		// e.g., "project/register" -> POST /api/project/register
		route = mcpToolRoute{
			Method:       "POST",
			Path:         "/api/" + toolName,
			BodyFromArgs: true,
		}
	}

	httpPayload := map[string]any{
		"http_method": route.Method,
	}

	path := route.Path

	// Handle path parameter extraction
	if route.PathParam != "" {
		if paramVal, ok := args[route.PathParam]; ok {
			path = path + "/" + fmt.Sprintf("%v", paramVal)
		}
	}

	// Handle agent-specific sub-actions
	switch toolName {
	case "agent/stop":
		path = path + "/stop"
	case "agent/restart":
		path = path + "/restart"
	case "agent/send_message":
		path = path + "/message:send"
	}

	// Handle query parameters
	if len(route.QueryParams) > 0 {
		var queryParts []string
		for _, qp := range route.QueryParams {
			if val, ok := args[qp]; ok {
				queryParts = append(queryParts, fmt.Sprintf("%s=%v", qp, val))
			}
		}
		if len(queryParts) > 0 {
			path = path + "?" + strings.Join(queryParts, "&")
		}
	}

	httpPayload["path"] = path

	// Set body for POST/PUT/DELETE with body
	if route.BodyFromArgs && len(args) > 0 {
		// Remove path param from body if it was used in the URL
		body := make(map[string]any, len(args))
		for k, v := range args {
			body[k] = v
		}
		httpPayload["body"] = body
		httpPayload["headers"] = map[string]any{
			"Content-Type": "application/json",
		}
	}

	// For DELETE with body (like agent/stop with grace_period_ms)
	if route.Method == "POST" && route.PathParam != "" && len(args) > 1 {
		body := make(map[string]any)
		for k, v := range args {
			if k != route.PathParam {
				body[k] = v
			}
		}
		if len(body) > 0 {
			httpPayload["body"] = body
			httpPayload["headers"] = map[string]any{
				"Content-Type": "application/json",
			}
		}
	}

	return &schema.Step{
		Action:  schema.ActionSend,
		Payload: httpPayload,
	}
}
