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
			Timeout: 30 * time.Second,
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
	if b, ok := payload["body"]; ok {
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
	}

	t.lastResp = tr
	return tr, nil
}
