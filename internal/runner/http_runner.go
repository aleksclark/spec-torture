package runner

import (
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

// HTTPTransport implements the Transport interface for plain HTTP REST APIs.
type HTTPTransport struct {
	baseURL  string
	client   *http.Client
	logger   *slog.Logger
	lastResp *TransportResponse
}

// NewHTTPTransport creates a new HTTP REST transport.
func NewHTTPTransport(baseURL string, logger *slog.Logger) *HTTPTransport {
	return &HTTPTransport{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		logger: logger,
	}
}

// LastResponse returns the most recent transport response.
func (t *HTTPTransport) LastResponse() *TransportResponse {
	return t.lastResp
}

// Send executes a send step via plain HTTP.
func (t *HTTPTransport) Send(ctx context.Context, step *schema.Step) (*TransportResponse, error) {
	payload := step.Payload
	if t.lastResp != nil {
		payload = interpolateVars(payload, t.lastResp)
	}

	method := "GET"
	if m, ok := payload["http_method"]; ok {
		method = fmt.Sprintf("%v", m)
	}

	path := ""
	if p, ok := payload["path"]; ok {
		path = fmt.Sprintf("%v", p)
	}

	url := t.baseURL + path
	t.logger.Debug("sending HTTP request", "method", method, "url", url)

	var bodyReader io.Reader
	if rawBody, ok := payload["raw_body"]; ok {
		// raw_body is sent as-is (for malformed JSON tests)
		bodyReader = strings.NewReader(fmt.Sprintf("%v", rawBody))
	} else if b, ok := payload["body"]; ok {
		bodyBytes, err := json.Marshal(b)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if headers, ok := payload["headers"]; ok {
		if hm, ok := headers.(map[string]any); ok {
			for k, v := range hm {
				req.Header.Set(k, fmt.Sprintf("%v", v))
			}
		}
	}

	// Check for polling config
	if pollCfg, hasPoll := payload["poll"]; hasPoll {
		return t.executePollingSend(ctx, method, url, payload, pollCfg)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
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

	var jsonBody map[string]any
	if err := json.Unmarshal(body, &jsonBody); err == nil {
		tr.Body = jsonBody
	} else {
		var jsonArray []any
		if err := json.Unmarshal(body, &jsonArray); err == nil {
			tr.BodyArray = jsonArray
		}
	}

	t.lastResp = tr
	return tr, nil
}

// executePollingSend handles send steps with poll configuration.
func (t *HTTPTransport) executePollingSend(ctx context.Context, method, url string, payload map[string]any, pollCfg any) (*TransportResponse, error) {
	pollMap, ok := pollCfg.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("poll config must be a map with interval and max_attempts")
	}

	intervalStr := "2s"
	if is, ok := pollMap["interval"]; ok {
		intervalStr = fmt.Sprintf("%v", is)
	}
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		interval = 2 * time.Second
	}

	maxAttempts := 10
	if ma, ok := pollMap["max_attempts"]; ok {
		switch v := ma.(type) {
		case int:
			maxAttempts = v
		case float64:
			maxAttempts = int(v)
		}
	}

	var lastResp *TransportResponse
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		t.logger.Debug("polling attempt", "attempt", attempt, "max", maxAttempts, "url", url)

		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating poll request: %w", err)
		}

		if headers, ok := payload["headers"]; ok {
			if hm, ok := headers.(map[string]any); ok {
				for k, v := range hm {
					req.Header.Set(k, fmt.Sprintf("%v", v))
				}
			}
		}

		resp, err := t.client.Do(req)
		if err != nil {
			if attempt == maxAttempts {
				return nil, fmt.Errorf("HTTP request failed on poll attempt %d: %w", attempt, err)
			}
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during polling")
			case <-time.After(interval):
				continue
			}
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		tr := &TransportResponse{
			HTTPStatus: resp.StatusCode,
			Headers:    normalizeHeaders(resp.Header),
			RawBody:    string(body),
		}

		var jsonBody map[string]any
		if err := json.Unmarshal(body, &jsonBody); err == nil {
			tr.Body = jsonBody
		} else {
			var jsonArray []any
			if err := json.Unmarshal(body, &jsonArray); err == nil {
				tr.BodyArray = jsonArray
			}
		}

		lastResp = tr
		t.lastResp = tr

		// If we have a JSON body with a "status" field, check if it's a terminal state
		if tr.Body != nil {
			if status, ok := tr.Body["status"]; ok {
				statusStr := fmt.Sprintf("%v", status)
				if statusStr == "completed" || statusStr == "failed" || statusStr == "cancelled" ||
					statusStr == "ready" || statusStr == "stopped" || statusStr == "error" {
					return tr, nil
				}
			}
		}

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during polling")
			case <-time.After(interval):
			}
		}
	}

	// If we exhausted all attempts, still return the last response
	// (the expect step will check the actual values)
	return lastResp, nil
}
