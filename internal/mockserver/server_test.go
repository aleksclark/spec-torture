package mockserver_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aleksclark/spec-torture/internal/mockserver"
)

func TestAgentCard(t *testing.T) {
	s, err := mockserver.New()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	resp, err := http.Get(s.URL() + "/.well-known/agent-card.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var card map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		t.Fatal(err)
	}

	if card["name"] != "mock-a2a-agent" {
		t.Fatalf("expected name mock-a2a-agent, got %v", card["name"])
	}

	reqs := s.Requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 recorded request, got %d", len(reqs))
	}
	if reqs[0].Path != "/.well-known/agent-card.json" {
		t.Fatalf("expected path /.well-known/agent-card.json, got %s", reqs[0].Path)
	}
}

func TestSendMessage(t *testing.T) {
	s, err := mockserver.New()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"message":{"messageId":"test-001","role":"ROLE_USER","parts":[{"text":"hi"}]}}}`
	resp, err := http.Post(s.URL()+"/", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("bad JSON: %s", respBody)
	}

	if result["jsonrpc"] != "2.0" {
		t.Fatal("missing jsonrpc 2.0")
	}
	if result["error"] != nil {
		t.Fatalf("unexpected error: %v", result["error"])
	}

	reqs := s.RequestsByMethod("SendMessage")
	if len(reqs) != 1 {
		t.Fatalf("expected 1 SendMessage request, got %d", len(reqs))
	}
}

func TestGetTaskNotFound(t *testing.T) {
	s, err := mockserver.New()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"GetTask","params":{"id":"nonexistent"}}`
	resp, err := http.Post(s.URL()+"/", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(respBody, &result)

	errObj, _ := result["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error for nonexistent task")
	}
	code, _ := errObj["code"].(float64)
	if code != -32001 {
		t.Fatalf("expected error code -32001, got %v", code)
	}
}

func TestRecordedRequests(t *testing.T) {
	s, err := mockserver.New()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"SendMessage","params":{"message":{"messageId":"test-001","role":"ROLE_USER","parts":[{"text":"hi"}]}}}`
	http.Post(s.URL()+"/", "application/json", strings.NewReader(body))

	body2 := `{"jsonrpc":"2.0","id":2,"method":"GetTask","params":{"id":"task-mock-001"}}`
	http.Post(s.URL()+"/", "application/json", strings.NewReader(body2))

	all := s.Requests()
	if len(all) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(all))
	}

	sends := s.RequestsByMethod("SendMessage")
	if len(sends) != 1 {
		t.Fatalf("expected 1 SendMessage, got %d", len(sends))
	}

	gets := s.RequestsByMethod("GetTask")
	if len(gets) != 1 {
		t.Fatalf("expected 1 GetTask, got %d", len(gets))
	}

	s.Reset()
	if len(s.Requests()) != 0 {
		t.Fatal("expected 0 requests after reset")
	}
}
