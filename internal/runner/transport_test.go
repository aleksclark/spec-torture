package runner

import (
	"testing"
)

func TestResolveJSONPath_MapOnly(t *testing.T) {
	data := map[string]any{
		"result": map[string]any{
			"id":   "task-123",
			"name": "test",
		},
	}
	got := resolveJSONPath(data, "result.id")
	if got != "task-123" {
		t.Fatalf("expected task-123, got %v", got)
	}
}

func TestResolveJSONPath_ArrayIndex(t *testing.T) {
	data := map[string]any{
		"body": []any{
			map[string]any{
				"metadata": map[string]any{
					"agent_id": "agent-1",
				},
				"name": "first",
			},
			map[string]any{
				"name": "second",
			},
		},
	}

	got := resolveJSONPath(data, "body.0.name")
	if got != "first" {
		t.Fatalf("expected first, got %v", got)
	}

	got = resolveJSONPath(data, "body.0.metadata.agent_id")
	if got != "agent-1" {
		t.Fatalf("expected agent-1, got %v", got)
	}

	got = resolveJSONPath(data, "body.1.name")
	if got != "second" {
		t.Fatalf("expected second, got %v", got)
	}
}

func TestResolveJSONPath_ArrayOutOfBounds(t *testing.T) {
	data := map[string]any{
		"body": []any{"a", "b"},
	}
	got := resolveJSONPath(data, "body.5")
	if got != nil {
		t.Fatalf("expected nil for out-of-bounds, got %v", got)
	}

	got = resolveJSONPath(data, "body.-1")
	if got != nil {
		t.Fatalf("expected nil for negative index, got %v", got)
	}
}

func TestResolveJSONPath_NonIntOnArray(t *testing.T) {
	data := map[string]any{
		"body": []any{"a"},
	}
	got := resolveJSONPath(data, "body.foo")
	if got != nil {
		t.Fatalf("expected nil for non-int array key, got %v", got)
	}
}

func TestMergeResponseData_MapBody(t *testing.T) {
	resp := &TransportResponse{
		Body: map[string]any{"id": "123"},
	}
	merged := mergeResponseData(resp)
	if merged["id"] != "123" {
		t.Fatalf("expected id=123, got %v", merged["id"])
	}
}

func TestMergeResponseData_ArrayBody(t *testing.T) {
	resp := &TransportResponse{
		BodyArray: []any{
			map[string]any{"name": "agent-1"},
			map[string]any{"name": "agent-2"},
		},
	}
	merged := mergeResponseData(resp)
	body, ok := merged["body"]
	if !ok {
		t.Fatal("expected 'body' key in merged data for array response")
	}
	arr, ok := body.([]any)
	if !ok {
		t.Fatal("expected body to be []any")
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr))
	}
	first := arr[0].(map[string]any)
	if first["name"] != "agent-1" {
		t.Fatalf("expected agent-1, got %v", first["name"])
	}
}

func TestMergeResponseData_JSONRPCTakesPrecedence(t *testing.T) {
	resp := &TransportResponse{
		JSONRPCResp: map[string]any{"result": "ok"},
		Body:        map[string]any{"id": "123"},
		BodyArray:   []any{"ignored"},
	}
	merged := mergeResponseData(resp)
	if merged["result"] != "ok" {
		t.Fatalf("expected JSONRPC to take precedence, got %v", merged)
	}
}

func TestInterpolateVars_MapBody(t *testing.T) {
	prev := &TransportResponse{
		Body: map[string]any{
			"result": map[string]any{"id": "task-42"},
		},
	}
	payload := map[string]any{
		"task_id": "$prev.result.id",
	}
	out := interpolateVars(payload, prev)
	if out["task_id"] != "task-42" {
		t.Fatalf("expected task-42, got %v", out["task_id"])
	}
}

func TestInterpolateVars_ArrayBody(t *testing.T) {
	prev := &TransportResponse{
		BodyArray: []any{
			map[string]any{
				"metadata": map[string]any{
					"arp": map[string]any{
						"agent_id": "agent-007",
					},
				},
			},
		},
	}
	payload := map[string]any{
		"agent": "$prev.body.0.metadata.arp.agent_id",
	}
	out := interpolateVars(payload, prev)
	if out["agent"] != "agent-007" {
		t.Fatalf("expected agent-007, got %v", out["agent"])
	}
}

func TestInterpolateVars_NilPrev(t *testing.T) {
	payload := map[string]any{"x": "$prev.foo"}
	out := interpolateVars(payload, nil)
	if out["x"] != "$prev.foo" {
		t.Fatalf("expected unchanged, got %v", out["x"])
	}
}
