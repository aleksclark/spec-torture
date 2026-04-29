package runner

import (
	"log/slog"
	"strings"
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

func newTestEvaluator() *Evaluator {
	return NewEvaluator(slog.Default())
}

func TestMatchExpect_BodyWildcard_MapResponse(t *testing.T) {
	eval := newTestEvaluator()
	resp := &TransportResponse{
		Body: map[string]any{"id": "123"},
	}
	expect := map[string]any{"body": "*"}
	if err := matchExpect(eval, resp, expect); err != nil {
		t.Fatalf("body wildcard should match map response: %v", err)
	}
}

func TestMatchExpect_BodyWildcard_ArrayResponse(t *testing.T) {
	eval := newTestEvaluator()
	resp := &TransportResponse{
		BodyArray: []any{"a", "b"},
	}
	expect := map[string]any{"body": "*"}
	if err := matchExpect(eval, resp, expect); err != nil {
		t.Fatalf("body wildcard should match array response: %v", err)
	}
}

func TestMatchExpect_BodyWildcard_RawBodyResponse(t *testing.T) {
	eval := newTestEvaluator()
	resp := &TransportResponse{
		RawBody: "hello",
	}
	expect := map[string]any{"body": "*"}
	if err := matchExpect(eval, resp, expect); err != nil {
		t.Fatalf("body wildcard should match raw body response: %v", err)
	}
}

func TestMatchExpect_BodyWildcard_NoBody(t *testing.T) {
	eval := newTestEvaluator()
	resp := &TransportResponse{}
	expect := map[string]any{"body": "*"}
	err := matchExpect(eval, resp, expect)
	if err == nil {
		t.Fatal("body wildcard should fail when response has no body")
	}
	if !strings.Contains(err.Error(), "expected body but response has no body") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestMatchExpect_BodyMap_StillWorks(t *testing.T) {
	eval := newTestEvaluator()
	resp := &TransportResponse{
		Body: map[string]any{"id": "123", "name": "test"},
	}
	expect := map[string]any{
		"body": map[string]any{"id": "123"},
	}
	if err := matchExpect(eval, resp, expect); err != nil {
		t.Fatalf("body map matching should still work: %v", err)
	}
}

func TestMatchExpect_BodyInvalidType(t *testing.T) {
	eval := newTestEvaluator()
	resp := &TransportResponse{
		Body: map[string]any{"id": "123"},
	}
	expect := map[string]any{"body": 42}
	err := matchExpect(eval, resp, expect)
	if err == nil {
		t.Fatal("body with invalid type should fail")
	}
	if !strings.Contains(err.Error(), "expected body must be a map") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
