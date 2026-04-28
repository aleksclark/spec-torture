package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/aleksclark/spec-torture/internal/schema"
)

// TransportResponse captures the result of a transport step execution.
type TransportResponse struct {
	HTTPStatus  int                    `json:"http_status,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Body        map[string]any         `json:"body,omitempty"`
	BodyArray   []any                  `json:"body_array,omitempty"`
	RawBody     string                 `json:"raw_body,omitempty"`
	SSEEvents   []map[string]any       `json:"sse_events,omitempty"`
	JSONRPCResp map[string]any         `json:"jsonrpc_resp,omitempty"`
	Parsed      map[string]any         `json:"parsed,omitempty"`
}

// Transport is the interface for protocol-specific step execution.
type Transport interface {
	// Send executes a send step and stores the response.
	Send(ctx context.Context, step *schema.Step) (*TransportResponse, error)
	// LastResponse returns the most recent response for variable interpolation.
	LastResponse() *TransportResponse
}

// varRefRE matches variable references like $prev.result.id
var varRefRE = regexp.MustCompile(`\$prev\.([a-zA-Z0-9_.]+)`)

// interpolateVars replaces $prev.field.path references in a payload map
// using the previous response data.
func interpolateVars(payload map[string]any, prev *TransportResponse) map[string]any {
	if prev == nil {
		return payload
	}
	merged := mergeResponseData(prev)
	return interpolateMap(payload, merged)
}

func mergeResponseData(resp *TransportResponse) map[string]any {
	if resp == nil {
		return nil
	}
	// Prefer JSONRPCResp, fall back to Body
	if resp.JSONRPCResp != nil {
		return resp.JSONRPCResp
	}
	if resp.Body != nil {
		return resp.Body
	}
	if resp.BodyArray != nil {
		return map[string]any{"body": resp.BodyArray}
	}
	// Try to get from SSE events (last event)
	if len(resp.SSEEvents) > 0 {
		return resp.SSEEvents[len(resp.SSEEvents)-1]
	}
	return nil
}

func interpolateMap(m map[string]any, data map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = interpolateValue(v, data)
	}
	return result
}

func interpolateValue(v any, data map[string]any) any {
	switch val := v.(type) {
	case string:
		if varRefRE.MatchString(val) {
			return varRefRE.ReplaceAllStringFunc(val, func(match string) string {
				path := match[len("$prev."):]
				resolved := resolveJSONPath(data, path)
				return fmt.Sprintf("%v", resolved)
			})
		}
		return val
	case map[string]any:
		return interpolateMap(val, data)
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = interpolateValue(item, data)
		}
		return out
	default:
		return v
	}
}

func resolveJSONPath(data map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = data
	for _, part := range parts {
		switch m := current.(type) {
		case map[string]any:
			current = m[part]
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(m) {
				return nil
			}
			current = m[idx]
		default:
			return nil
		}
	}
	return current
}

// matchExpect checks a TransportResponse against an expect step.
func matchExpect(eval *Evaluator, resp *TransportResponse, expect map[string]any) error {
	if resp == nil {
		return fmt.Errorf("no response to match against")
	}

	// Check http_status
	if expStatus, ok := expect["http_status"]; ok {
		expInt := toInt(expStatus)
		if resp.HTTPStatus != expInt {
			return fmt.Errorf("expected HTTP status %d, got %d", expInt, resp.HTTPStatus)
		}
	}

	// Check headers
	if expHeaders, ok := expect["headers"]; ok {
		expMap, ok := expHeaders.(map[string]any)
		if !ok {
			return fmt.Errorf("expected headers must be a map")
		}
		for k, v := range expMap {
			actual, exists := resp.Headers[strings.ToLower(k)]
			if !exists {
				return fmt.Errorf("missing expected header: %s", k)
			}
			expStr := fmt.Sprintf("%v", v)
			if expStr != "*" && !strings.Contains(strings.ToLower(actual), strings.ToLower(expStr)) {
				return fmt.Errorf("header %s: expected %q, got %q", k, expStr, actual)
			}
		}
	}

	// Check body (partial match against response body)
	if expBody, ok := expect["body"]; ok {
		// Wildcard: body: "*" means just check that a body exists
		if expStr, ok := expBody.(string); ok && expStr == "*" {
			if resp.Body == nil && resp.BodyArray == nil && resp.RawBody == "" {
				return fmt.Errorf("expected body but response has no body")
			}
			// Wildcard matches — body exists, skip detailed matching
		} else {
			expMap, ok := expBody.(map[string]any)
			if !ok {
				return fmt.Errorf("expected body must be a map or \"*\" wildcard")
			}
			if resp.Body == nil {
				return fmt.Errorf("response has no body to match against")
			}
			if err := eval.Match(expMap, resp.Body); err != nil {
				return fmt.Errorf("body match failed: %w", err)
			}
		}
	}

	// Check JSON-RPC fields (jsonrpc, id, result, error)
	if _, ok := expect["jsonrpc"]; ok {
		if resp.JSONRPCResp == nil {
			return fmt.Errorf("expected JSON-RPC response but got none")
		}
		if err := eval.Match(expect, resp.JSONRPCResp); err != nil {
			return fmt.Errorf("JSON-RPC match failed: %w", err)
		}
	}

	// Check SSE events
	if expSSE, ok := expect["sse_event"]; ok {
		if len(resp.SSEEvents) == 0 {
			return fmt.Errorf("expected SSE events but got none")
		}
		// Wildcard "*" just checks that at least one event was received
		if s, ok := expSSE.(string); ok && s == "*" {
			return nil
		}
		expMap, ok := expSSE.(map[string]any)
		if !ok {
			return fmt.Errorf("expected sse_event must be a map or \"*\" wildcard")
		}
		// Check if any SSE event matches
		var lastErr error
		for _, event := range resp.SSEEvents {
			if err := eval.Match(expMap, event); err != nil {
				lastErr = err
				continue
			}
			return nil
		}
		return fmt.Errorf("no SSE event matched: %w", lastErr)
	}

	return nil
}

func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case json.Number:
		n, _ := val.Int64()
		return int(n)
	default:
		return 0
	}
}
