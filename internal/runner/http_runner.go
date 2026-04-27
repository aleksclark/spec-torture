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

// HTTPTransport executes test steps against an HTTP REST server.
type HTTPTransport struct {
	baseURL  string
	client   *http.Client
	eval     *Evaluator
	logger   *slog.Logger
	vars     map[string]any // extracted variables from responses
	lastResp *httpResponse  // last HTTP response received
}

type httpResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	BodyJSON   map[string]any // parsed JSON body (nil if not JSON)
	NDJSONEvents []map[string]any // parsed NDJSON lines (nil if not NDJSON)
}

// NewHTTPTransport creates an HTTPTransport targeting the given base URL.
func NewHTTPTransport(baseURL string, eval *Evaluator, logger *slog.Logger) *HTTPTransport {
	return &HTTPTransport{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		eval:   eval,
		logger: logger,
		vars:   make(map[string]any),
	}
}

// Reset clears stored state between test cases.
func (h *HTTPTransport) Reset() {
	h.vars = make(map[string]any)
	h.lastResp = nil
}

// ExecuteStep runs a single test step and returns the result.
func (h *HTTPTransport) ExecuteStep(ctx context.Context, step *schema.Step, index int) schema.StepResult {
	start := time.Now()
	sr := schema.StepResult{
		StepIndex: index,
		Action:    step.Action,
	}

	h.logger.Debug("executing HTTP step", "index", index, "action", step.Action)

	switch step.Action {
	case schema.ActionSend:
		sr = h.executeSend(ctx, step, index)
	case schema.ActionExpect:
		sr = h.executeExpect(step, index)
	case schema.ActionWait:
		sr = h.executeWait(ctx, step, index)
	default:
		sr.Status = schema.StatusError
		sr.Error = fmt.Sprintf("unsupported action for http-rest: %s", step.Action)
	}

	if sr.Duration == 0 {
		sr.Duration = time.Since(start)
	}
	return sr
}

func (h *HTTPTransport) executeSend(ctx context.Context, step *schema.Step, index int) schema.StepResult {
	start := time.Now()
	sr := schema.StepResult{StepIndex: index, Action: step.Action}

	payload := step.Payload
	method, _ := toString(payload["http_method"])
	path, _ := toString(payload["path"])

	if method == "" {
		sr.Status = schema.StatusError
		sr.Error = "send step missing http_method"
		sr.Duration = time.Since(start)
		return sr
	}
	if path == "" {
		sr.Status = schema.StatusError
		sr.Error = "send step missing path"
		sr.Duration = time.Since(start)
		return sr
	}

	// Expand variables in the path: {run_id} -> value from vars
	path = h.expandVars(path)

	url := h.baseURL + path

	// Build request body
	var bodyReader io.Reader
	if rawBody, ok := payload["raw_body"]; ok {
		// raw_body is sent as-is (for malformed JSON tests)
		s, _ := toString(rawBody)
		bodyReader = strings.NewReader(s)
	} else if body, ok := payload["body"]; ok {
		// body_from_previous: use the last response body as the POST body
		if bfp, ok := payload["body_from_previous"]; ok {
			if bfpBool, isBool := bfp.(bool); isBool && bfpBool && h.lastResp != nil {
				bodyReader = bytes.NewReader(h.lastResp.Body)
			}
		}
		if bodyReader == nil {
			jsonBytes, err := json.Marshal(body)
			if err != nil {
				sr.Status = schema.StatusError
				sr.Error = fmt.Sprintf("marshaling request body: %v", err)
				sr.Duration = time.Since(start)
				return sr
			}
			bodyReader = bytes.NewReader(jsonBytes)
		}
	}

	// Check for body_from_previous at top level (not nested under body)
	if bodyReader == nil {
		if bfp, ok := payload["body_from_previous"]; ok {
			if bfpBool, isBool := bfp.(bool); isBool && bfpBool && h.lastResp != nil {
				bodyReader = bytes.NewReader(h.lastResp.Body)
			}
		}
	}

	// Check for polling config
	pollCfg, hasPoll := payload["poll"]
	if hasPoll {
		return h.executePollingSend(ctx, step, index, method, url, bodyReader, payload, pollCfg, start)
	}

	resp, err := h.doRequest(ctx, method, url, bodyReader, payload)
	if err != nil {
		sr.Status = schema.StatusError
		sr.Error = fmt.Sprintf("HTTP request failed: %v", err)
		sr.Duration = time.Since(start)
		return sr
	}

	h.storeResponse(resp)

	sr.Status = schema.StatusPass
	sr.Duration = time.Since(start)
	return sr
}

func (h *HTTPTransport) executePollingSend(ctx context.Context, step *schema.Step, index int, method, url string, bodyReader io.Reader, payload map[string]any, pollCfg any, start time.Time) schema.StepResult {
	sr := schema.StepResult{StepIndex: index, Action: step.Action}

	pollMap, ok := pollCfg.(map[string]any)
	if !ok {
		sr.Status = schema.StatusError
		sr.Error = "poll config must be a map with interval and max_attempts"
		sr.Duration = time.Since(start)
		return sr
	}

	intervalStr, _ := toString(pollMap["interval"])
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

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		h.logger.Debug("polling attempt", "attempt", attempt, "max", maxAttempts, "url", url)

		resp, err := h.doRequest(ctx, method, url, nil, payload)
		if err != nil {
			if attempt == maxAttempts {
				sr.Status = schema.StatusError
				sr.Error = fmt.Sprintf("HTTP request failed on poll attempt %d: %v", attempt, err)
				sr.Duration = time.Since(start)
				return sr
			}
			select {
			case <-ctx.Done():
				sr.Status = schema.StatusTimeout
				sr.Error = "context cancelled during polling"
				sr.Duration = time.Since(start)
				return sr
			case <-time.After(interval):
				continue
			}
		}

		h.storeResponse(resp)

		// If we have a JSON body with a "status" field, check if it's a terminal state
		if resp.BodyJSON != nil {
			if status, ok := resp.BodyJSON["status"]; ok {
				statusStr, _ := toString(status)
				if statusStr == "completed" || statusStr == "failed" || statusStr == "cancelled" {
					sr.Status = schema.StatusPass
					sr.Duration = time.Since(start)
					return sr
				}
			}
		}

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				sr.Status = schema.StatusTimeout
				sr.Error = "context cancelled during polling"
				sr.Duration = time.Since(start)
				return sr
			case <-time.After(interval):
			}
		}
	}

	// If we exhausted all attempts, still store the last response and pass
	// (the expect step will check the actual values)
	sr.Status = schema.StatusPass
	sr.Duration = time.Since(start)
	return sr
}

func (h *HTTPTransport) doRequest(ctx context.Context, method, url string, body io.Reader, payload map[string]any) (*httpResponse, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers from payload
	if hdrs, ok := payload["headers"]; ok {
		if hdrMap, ok := hdrs.(map[string]any); ok {
			for k, v := range hdrMap {
				s, _ := toString(v)
				req.Header.Set(k, s)
			}
		}
	}

	h.logger.Debug("sending HTTP request", "method", method, "url", url)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	result := &httpResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       bodyBytes,
	}

	ct := resp.Header.Get("Content-Type")

	// Try to parse as NDJSON if content type suggests it
	if strings.Contains(ct, "ndjson") || strings.Contains(ct, "x-ndjson") {
		result.NDJSONEvents = parseNDJSON(bodyBytes)
		// Also try to parse the first event as JSON for compatibility
		if len(result.NDJSONEvents) > 0 {
			result.BodyJSON = result.NDJSONEvents[0]
		}
	} else {
		// Try to parse as JSON
		var parsed map[string]any
		if err := json.Unmarshal(bodyBytes, &parsed); err == nil {
			result.BodyJSON = parsed
		}
	}

	h.logger.Debug("received HTTP response",
		"status", resp.StatusCode,
		"content_type", ct,
		"body_len", len(bodyBytes),
	)

	return result, nil
}

func (h *HTTPTransport) storeResponse(resp *httpResponse) {
	h.lastResp = resp

	// Extract variables from JSON response for use in subsequent steps
	if resp.BodyJSON != nil {
		for k, v := range resp.BodyJSON {
			h.vars[k] = v
		}
	}
}

func (h *HTTPTransport) executeExpect(step *schema.Step, index int) schema.StepResult {
	start := time.Now()
	sr := schema.StepResult{StepIndex: index, Action: step.Action}

	if h.lastResp == nil {
		sr.Status = schema.StatusError
		sr.Error = "expect step with no prior response"
		sr.Duration = time.Since(start)
		return sr
	}

	expect := step.Expect

	// Check HTTP status
	if expStatus, ok := expect["http_status"]; ok {
		expectedCode := toInt(expStatus)
		if expectedCode != h.lastResp.StatusCode {
			sr.Status = schema.StatusFail
			sr.Expected = expectedCode
			sr.Actual = h.lastResp.StatusCode
			sr.Error = fmt.Sprintf("expected HTTP %d, got HTTP %d", expectedCode, h.lastResp.StatusCode)
			sr.Duration = time.Since(start)
			return sr
		}
	}

	// Check response headers
	if expHeaders, ok := expect["headers"]; ok {
		if hdrMap, ok := expHeaders.(map[string]any); ok {
			for key, expVal := range hdrMap {
				expStr, _ := toString(expVal)
				actualVal := h.lastResp.Headers.Get(key)
				if actualVal == "" {
					sr.Status = schema.StatusFail
					sr.Expected = fmt.Sprintf("header %s: %s", key, expStr)
					sr.Actual = fmt.Sprintf("header %s: (missing)", key)
					sr.Error = fmt.Sprintf("expected header %s to be present", key)
					sr.Duration = time.Since(start)
					return sr
				}
				if !matchGlob(expStr, actualVal) {
					sr.Status = schema.StatusFail
					sr.Expected = fmt.Sprintf("header %s: %s", key, expStr)
					sr.Actual = fmt.Sprintf("header %s: %s", key, actualVal)
					sr.Error = fmt.Sprintf("header %s mismatch: expected %q, got %q", key, expStr, actualVal)
					sr.Duration = time.Since(start)
					return sr
				}
			}
		}
	}

	// Check body_format
	if bodyFmt, ok := expect["body_format"]; ok {
		fmtStr, _ := toString(bodyFmt)
		if fmtStr == "ndjson" {
			if h.lastResp.NDJSONEvents == nil || len(h.lastResp.NDJSONEvents) == 0 {
				sr.Status = schema.StatusFail
				sr.Expected = "NDJSON body"
				sr.Actual = string(h.lastResp.Body)
				sr.Error = "expected NDJSON body but got none"
				sr.Duration = time.Since(start)
				return sr
			}
		}
	}

	// Check ndjson_contains: verify that certain events appear in the stream
	if ndjsonContains, ok := expect["ndjson_contains"]; ok {
		if patterns, ok := ndjsonContains.([]any); ok {
			events := h.lastResp.NDJSONEvents
			if events == nil {
				sr.Status = schema.StatusFail
				sr.Expected = "NDJSON events"
				sr.Actual = string(h.lastResp.Body)
				sr.Error = "expected NDJSON events but response is not NDJSON"
				sr.Duration = time.Since(start)
				return sr
			}
			for _, pattern := range patterns {
				patMap, ok := pattern.(map[string]any)
				if !ok {
					continue
				}
				if !containsMatchingEvent(events, patMap) {
					sr.Status = schema.StatusFail
					sr.Expected = patMap
					sr.Error = fmt.Sprintf("NDJSON stream does not contain event matching %v", patMap)
					sr.Duration = time.Since(start)
					return sr
				}
			}
		}
	}

	// Check ndjson_last: the last NDJSON event matches the pattern
	if ndjsonLast, ok := expect["ndjson_last"]; ok {
		if patMap, ok := ndjsonLast.(map[string]any); ok {
			events := h.lastResp.NDJSONEvents
			if events == nil || len(events) == 0 {
				sr.Status = schema.StatusFail
				sr.Expected = "NDJSON last event"
				sr.Error = "no NDJSON events to check last"
				sr.Duration = time.Since(start)
				return sr
			}
			lastEvent := events[len(events)-1]
			if err := matchMapPartial(patMap, lastEvent); err != nil {
				sr.Status = schema.StatusFail
				sr.Expected = patMap
				sr.Actual = lastEvent
				sr.Error = fmt.Sprintf("last NDJSON event mismatch: %v", err)
				sr.Duration = time.Since(start)
				return sr
			}
		}
	}

	// Check body (string or map)
	if expBody, ok := expect["body"]; ok {
		switch exp := expBody.(type) {
		case string:
			// String body comparison
			actualBody := strings.TrimSpace(string(h.lastResp.Body))
			if !matchGlob(exp, actualBody) {
				sr.Status = schema.StatusFail
				sr.Expected = exp
				sr.Actual = actualBody
				sr.Error = fmt.Sprintf("body mismatch: expected %q, got %q", exp, actualBody)
				sr.Duration = time.Since(start)
				return sr
			}
		case map[string]any:
			// JSON body partial match
			if h.lastResp.BodyJSON == nil {
				sr.Status = schema.StatusFail
				sr.Expected = exp
				sr.Actual = string(h.lastResp.Body)
				sr.Error = "expected JSON body but response is not valid JSON"
				sr.Duration = time.Since(start)
				return sr
			}
			if err := h.eval.Match(exp, h.lastResp.BodyJSON); err != nil {
				sr.Status = schema.StatusFail
				sr.Expected = exp
				sr.Actual = h.lastResp.BodyJSON
				sr.Error = fmt.Sprintf("body match failed: %v", err)
				sr.Duration = time.Since(start)
				return sr
			}
		}
	}

	sr.Status = schema.StatusPass
	sr.Duration = time.Since(start)
	return sr
}

func (h *HTTPTransport) executeWait(ctx context.Context, step *schema.Step, index int) schema.StepResult {
	start := time.Now()
	sr := schema.StepResult{StepIndex: index, Action: step.Action}

	duration := step.Timeout
	if duration == 0 {
		// Check for duration in payload
		if d, ok := step.Payload["duration"]; ok {
			if ds, ok := d.(string); ok {
				parsed, err := time.ParseDuration(ds)
				if err == nil {
					duration = parsed
				}
			}
		}
	}
	if duration == 0 {
		duration = 1 * time.Second
	}

	h.logger.Debug("waiting", "duration", duration)

	select {
	case <-ctx.Done():
		sr.Status = schema.StatusTimeout
		sr.Error = "context cancelled during wait"
	case <-time.After(duration):
		sr.Status = schema.StatusPass
	}

	sr.Duration = time.Since(start)
	return sr
}

// expandVars replaces {var_name} placeholders in s with values from h.vars.
func (h *HTTPTransport) expandVars(s string) string {
	for k, v := range h.vars {
		placeholder := "{" + k + "}"
		if strings.Contains(s, placeholder) {
			s = strings.ReplaceAll(s, placeholder, fmt.Sprintf("%v", v))
		}
	}
	return s
}

// matchGlob does a simple glob match where * means "match anything".
// Only trailing * is supported: "text/plain*" matches "text/plain; charset=utf-8".
func matchGlob(pattern, actual string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(actual, prefix)
	}
	return pattern == actual
}

func parseNDJSON(data []byte) []map[string]any {
	var events []map[string]any
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err == nil {
			events = append(events, obj)
		}
	}
	return events
}

// containsMatchingEvent checks if any event in the list matches all fields of the pattern.
func containsMatchingEvent(events []map[string]any, pattern map[string]any) bool {
	for _, event := range events {
		if matchMapPartial(pattern, event) == nil {
			return true
		}
	}
	return false
}

// matchMapPartial checks that all keys in expected exist in actual with matching values.
// Uses glob matching for string values.
func matchMapPartial(expected, actual map[string]any) error {
	for key, expVal := range expected {
		actVal, ok := actual[key]
		if !ok {
			return fmt.Errorf("missing key %q", key)
		}
		if err := matchValuePartial(expVal, actVal); err != nil {
			return fmt.Errorf("key %q: %w", key, err)
		}
	}
	return nil
}

func matchValuePartial(expected, actual any) error {
	switch exp := expected.(type) {
	case string:
		if exp == "*" {
			return nil
		}
		actStr := fmt.Sprintf("%v", actual)
		if !matchGlob(exp, actStr) {
			return fmt.Errorf("expected %q, got %q", exp, actStr)
		}
	case map[string]any:
		actMap, ok := actual.(map[string]any)
		if !ok {
			return fmt.Errorf("expected map, got %T", actual)
		}
		return matchMapPartial(exp, actMap)
	case []any:
		actSlice, ok := actual.([]any)
		if !ok {
			return fmt.Errorf("expected array, got %T", actual)
		}
		if len(actSlice) < len(exp) {
			return fmt.Errorf("expected at least %d items, got %d", len(exp), len(actSlice))
		}
		for i, v := range exp {
			if err := matchValuePartial(v, actSlice[i]); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
	default:
		if fmt.Sprintf("%v", expected) != fmt.Sprintf("%v", actual) {
			return fmt.Errorf("expected %v, got %v", expected, actual)
		}
	}
	return nil
}

func toString(v any) (string, bool) {
	if v == nil {
		return "", false
	}
	switch s := v.(type) {
	case string:
		return s, true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	case string:
		var i int
		fmt.Sscanf(n, "%d", &i)
		return i
	default:
		return 0
	}
}
