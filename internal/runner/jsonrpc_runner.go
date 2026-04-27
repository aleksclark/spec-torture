package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aleksclark/spec-torture/internal/schema"
)

// JSONRPCTransport implements the Transport interface for JSON-RPC over HTTP.
type JSONRPCTransport struct {
	baseURL  string
	client   *http.Client
	logger   *slog.Logger
	lastResp *TransportResponse
}

// NewJSONRPCTransport creates a new JSON-RPC HTTP transport.
func NewJSONRPCTransport(baseURL string, logger *slog.Logger) *JSONRPCTransport {
	return &JSONRPCTransport{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// LastResponse returns the most recent transport response.
func (t *JSONRPCTransport) LastResponse() *TransportResponse {
	return t.lastResp
}

// Send executes a send step against the JSON-RPC HTTP endpoint.
func (t *JSONRPCTransport) Send(ctx context.Context, step *schema.Step) (*TransportResponse, error) {
	payload := step.Payload
	if t.lastResp != nil {
		payload = interpolateVars(payload, t.lastResp)
	}

	// Detect which kind of request this is
	if _, ok := payload["http_method"]; ok {
		return t.sendHTTP(ctx, payload)
	}
	if _, ok := payload["raw_body"]; ok {
		return t.sendRaw(ctx, payload)
	}
	if _, ok := payload["jsonrpc"]; ok {
		return t.sendJSONRPC(ctx, payload)
	}

	return nil, fmt.Errorf("cannot determine request type from payload keys: %v", mapKeys(payload))
}

// sendHTTP handles plain HTTP requests (like discovery endpoints).
func (t *JSONRPCTransport) sendHTTP(ctx context.Context, payload map[string]any) (*TransportResponse, error) {
	method := fmt.Sprintf("%v", payload["http_method"])
	path := ""
	if p, ok := payload["path"]; ok {
		path = fmt.Sprintf("%v", p)
	}

	url := t.baseURL + path
	t.logger.Debug("sending HTTP request", "method", method, "url", url)

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	tr := &TransportResponse{
		HTTPStatus: resp.StatusCode,
		Headers:    normalizeHeaders(resp.Header),
		RawBody:    string(body),
	}

	// Try to parse body as JSON
	var jsonBody map[string]any
	if err := json.Unmarshal(body, &jsonBody); err == nil {
		tr.Body = jsonBody
	}

	t.lastResp = tr
	t.logger.Debug("HTTP response", "status", resp.StatusCode, "body_len", len(body))
	return tr, nil
}

// sendRaw sends a raw body (for error testing like malformed JSON).
func (t *JSONRPCTransport) sendRaw(ctx context.Context, payload map[string]any) (*TransportResponse, error) {
	rawBody := fmt.Sprintf("%v", payload["raw_body"])
	contentType := "application/json"
	if ct, ok := payload["content_type"]; ok {
		contentType = fmt.Sprintf("%v", ct)
	}

	url := t.baseURL
	t.logger.Debug("sending raw request", "url", url, "content_type", contentType)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("raw request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	tr := &TransportResponse{
		HTTPStatus: resp.StatusCode,
		Headers:    normalizeHeaders(resp.Header),
		RawBody:    string(body),
	}

	var jsonResp map[string]any
	if err := json.Unmarshal(body, &jsonResp); err == nil {
		if _, hasJSONRPC := jsonResp["jsonrpc"]; hasJSONRPC {
			tr.JSONRPCResp = jsonResp
		} else {
			tr.Body = jsonResp
		}
	}

	t.lastResp = tr
	return tr, nil
}

// sendJSONRPC sends a JSON-RPC request over HTTP POST.
func (t *JSONRPCTransport) sendJSONRPC(ctx context.Context, payload map[string]any) (*TransportResponse, error) {
	method, _ := payload["method"].(string)

	// Check for custom headers (e.g., authorization override)
	var customHeaders map[string]any
	if h, ok := payload["headers"]; ok {
		customHeaders, _ = h.(map[string]any)
	}

	// For streaming methods, use SSE handling
	if method == "SendStreamingMessage" || method == "SubscribeToTask" ||
		method == "message/stream" || method == "tasks/resubscribe" {
		return t.sendJSONRPCStreaming(ctx, payload, customHeaders)
	}

	// Strip non-JSON-RPC fields from payload before marshaling
	cleanPayload := make(map[string]any, len(payload))
	for k, v := range payload {
		if k != "headers" {
			cleanPayload[k] = v
		}
	}

	bodyBytes, err := json.Marshal(cleanPayload)
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON-RPC request: %w", err)
	}

	url := t.baseURL
	t.logger.Debug("sending JSON-RPC request", "method", method, "url", url)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if customHeaders != nil {
		for k, v := range customHeaders {
			val := fmt.Sprintf("%v", v)
			if val == "" {
				req.Header.Del(k)
			} else {
				req.Header.Set(k, val)
			}
		}
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("JSON-RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	tr := &TransportResponse{
		HTTPStatus: resp.StatusCode,
		Headers:    normalizeHeaders(resp.Header),
		RawBody:    string(body),
	}

	var jsonResp map[string]any
	if err := json.Unmarshal(body, &jsonResp); err == nil {
		if _, hasJSONRPC := jsonResp["jsonrpc"]; hasJSONRPC {
			tr.JSONRPCResp = jsonResp
		} else {
			tr.Body = jsonResp
		}
	}

	t.lastResp = tr
	t.logger.Debug("JSON-RPC response", "status", resp.StatusCode, "body_len", len(body))
	return tr, nil
}

// sendJSONRPCStreaming handles streaming JSON-RPC methods that return SSE.
func (t *JSONRPCTransport) sendJSONRPCStreaming(ctx context.Context, payload map[string]any, customHeaders map[string]any) (*TransportResponse, error) {
	// Remove headers from payload before marshaling
	cleanPayload := make(map[string]any, len(payload))
	for k, v := range payload {
		if k != "headers" {
			cleanPayload[k] = v
		}
	}

	bodyBytes, err := json.Marshal(cleanPayload)
	if err != nil {
		return nil, fmt.Errorf("marshaling streaming request: %w", err)
	}

	url := t.baseURL
	t.logger.Debug("sending streaming JSON-RPC request", "url", url)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	if customHeaders != nil {
		for k, v := range customHeaders {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	// Use a client without timeout for streaming
	streamClient := &http.Client{Timeout: 0}
	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("streaming request failed: %w", err)
	}

	tr := &TransportResponse{
		HTTPStatus: resp.StatusCode,
		Headers:    normalizeHeaders(resp.Header),
	}

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/event-stream") {
		events := parseSSEStream(resp.Body, 10*time.Second)
		resp.Body.Close()
		tr.SSEEvents = events
		// Store the last event as the primary response data for $prev references
		if len(events) > 0 {
			tr.Parsed = events[len(events)-1]
		}
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		tr.RawBody = string(body)
		var jsonResp map[string]any
		if err := json.Unmarshal(body, &jsonResp); err == nil {
			if _, hasJSONRPC := jsonResp["jsonrpc"]; hasJSONRPC {
				tr.JSONRPCResp = jsonResp
			} else {
				tr.Body = jsonResp
			}
		}
	}

	t.lastResp = tr
	return tr, nil
}

// parseSSEStream reads SSE events from a stream with a timeout.
func parseSSEStream(r io.Reader, timeout time.Duration) []map[string]any {
	var events []map[string]any

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(r)
		var currentData strings.Builder

		for scanner.Scan() {
			line := scanner.Text()

			if line == "" {
				// End of event
				if currentData.Len() > 0 {
					var event map[string]any
					if err := json.Unmarshal([]byte(currentData.String()), &event); err == nil {
						events = append(events, event)
					}
					currentData.Reset()
				}
				continue
			}

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				currentData.WriteString(data)
			}
		}

		// Handle last event if no trailing newline
		if currentData.Len() > 0 {
			var event map[string]any
			if err := json.Unmarshal([]byte(currentData.String()), &event); err == nil {
				events = append(events, event)
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(timeout):
	}

	return events
}

// normalizeHeaders lowercases header keys and joins values.
func normalizeHeaders(h http.Header) map[string]string {
	result := make(map[string]string, len(h))
	for k, v := range h {
		result[strings.ToLower(k)] = strings.Join(v, ", ")
	}
	return result
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
