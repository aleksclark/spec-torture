package clientrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aleksclark/spec-torture/internal/mockserver"
	"github.com/aleksclark/spec-torture/internal/schema"
	"github.com/google/uuid"
)

// ClientDriver is an interface for driving a client under test.
// Different clients (crush-a2a, etc.) implement this interface.
type ClientDriver interface {
	// Discovery tells the client to discover the agent at the given URL.
	// Returns the raw response from the client.
	Discovery(ctx context.Context, agentURL string) (map[string]any, error)

	// SendMessage tells the client to send a message to the agent.
	SendMessage(ctx context.Context, agentURL string, params map[string]any) (map[string]any, error)

	// GetTask tells the client to retrieve a task by ID.
	GetTask(ctx context.Context, agentURL string, taskID string) (map[string]any, error)

	// SendStreamingMessage tells the client to send a streaming message.
	SendStreamingMessage(ctx context.Context, agentURL string, params map[string]any) (map[string]any, error)

	// Close releases any resources.
	Close() error
}

// TestCase defines a client-side conformance test.
type TestCase struct {
	ID          string        `yaml:"id" json:"id"`
	Name        string        `yaml:"name" json:"name"`
	Description string        `yaml:"description" json:"description"`
	Severity    string        `yaml:"severity" json:"severity"`
	Category    string        `yaml:"category" json:"category"`
	Tags        []string      `yaml:"tags" json:"tags"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`

	// Run is the test function. It receives the mock server and client driver,
	// performs the test, and returns nil on success or an error describing the failure.
	Run func(ctx context.Context, mock *mockserver.Server, driver ClientDriver) error `yaml:"-" json:"-"`
}

// ClientDriverConfigurable is optionally implemented by drivers that need
// the mock server URL configured before tests run.
type ClientDriverConfigurable interface {
	ConfigureMockServer(mockURL string) error
}

// Runner executes client conformance tests.
type Runner struct {
	logger *slog.Logger
	mock   *mockserver.Server
	driver ClientDriver
	tests  []TestCase
}

// New creates a new client test runner.
func New(logger *slog.Logger, driver ClientDriver, tests []TestCase) (*Runner, error) {
	mock, err := mockserver.New()
	if err != nil {
		return nil, fmt.Errorf("starting mock server: %w", err)
	}

	// If the driver supports configuration, tell it about the mock server.
	if configurable, ok := driver.(ClientDriverConfigurable); ok {
		if err := configurable.ConfigureMockServer(mock.URL()); err != nil {
			mock.Close()
			return nil, fmt.Errorf("configuring driver for mock server: %w", err)
		}
	}

	return &Runner{
		logger: logger,
		mock:   mock,
		driver: driver,
		tests:  tests,
	}, nil
}

// MockURL returns the URL of the mock server.
func (r *Runner) MockURL() string {
	return r.mock.URL()
}

// Run executes all test cases and returns a TestRun result.
func (r *Runner) Run(ctx context.Context, specID, runtime string, tags []string) *schema.TestRun {
	run := &schema.TestRun{
		ID:        uuid.New().String(),
		SpecID:    specID,
		Runtime:   runtime,
		StartedAt: time.Now(),
	}

	r.logger.Info("starting client test run",
		"run_id", run.ID,
		"spec", specID,
		"runtime", runtime,
		"mock_url", r.mock.URL(),
	)

	for _, tc := range r.tests {
		if len(tags) > 0 && !matchesTags(tc.Tags, tags) {
			run.Results = append(run.Results, schema.TestResult{
				ID:         uuid.New().String(),
				SpecID:     specID,
				TestCaseID: tc.ID,
				Runtime:    runtime,
				Status:     schema.StatusSkip,
			})
			continue
		}

		result := r.runTest(ctx, tc, specID, runtime)
		run.Results = append(run.Results, result)
	}

	run.CompletedAt = time.Now()
	run.ComputeSummary()

	r.logger.Info("client test run complete",
		"run_id", run.ID,
		"passed", run.Summary.Passed,
		"failed", run.Summary.Failed,
		"errors", run.Summary.Errors,
		"compliance", fmt.Sprintf("%.1f%%", run.Summary.Compliance),
	)

	return run
}

func (r *Runner) runTest(ctx context.Context, tc TestCase, specID, runtime string) schema.TestResult {
	result := schema.TestResult{
		ID:         uuid.New().String(),
		SpecID:     specID,
		TestCaseID: tc.ID,
		Runtime:    runtime,
		StartedAt:  time.Now(),
	}

	timeout := tc.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	r.logger.Info("running client test", "test_case", tc.ID, "category", tc.Category)

	// Reset mock server state before each test
	r.mock.Reset()

	if tc.Run == nil {
		result.Status = schema.StatusError
		result.ErrorMessage = "test has no Run function"
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		return result
	}

	err := tc.Run(testCtx, r.mock, r.driver)
	if err != nil {
		result.Status = schema.StatusFail
		result.ErrorMessage = err.Error()
	} else {
		result.Status = schema.StatusPass
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	return result
}

// Close shuts down the mock server and client driver.
func (r *Runner) Close() error {
	if r.mock != nil {
		r.mock.Close()
	}
	if r.driver != nil {
		r.driver.Close()
	}
	return nil
}

func matchesTags(testTags, filterTags []string) bool {
	tagSet := make(map[string]bool, len(testTags))
	for _, t := range testTags {
		tagSet[t] = true
	}
	for _, ft := range filterTags {
		if tagSet[ft] {
			return true
		}
	}
	return false
}

// Helper: check that a request was recorded with the given method.
func RequireRequestMethod(mock *mockserver.Server, method string) error {
	reqs := mock.RequestsByMethod(method)
	if len(reqs) == 0 {
		return fmt.Errorf("expected at least one %s request, got none", method)
	}
	return nil
}

// Helper: check a JSON-RPC request body field.
func RequireRequestField(req mockserver.Request, path string, expected any) error {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(req.RawBody), &parsed); err != nil {
		return fmt.Errorf("failed to parse request body: %w", err)
	}

	actual := resolveJSONPath(parsed, path)
	if actual == nil {
		return fmt.Errorf("field %q not found in request", path)
	}

	if expected == "*" {
		return nil
	}

	expStr := fmt.Sprintf("%v", expected)
	actStr := fmt.Sprintf("%v", actual)
	if expStr != actStr {
		return fmt.Errorf("field %q: expected %v, got %v", path, expected, actual)
	}
	return nil
}

// Helper: validate that a request has proper JSON-RPC 2.0 envelope.
func RequireJSONRPCEnvelope(req mockserver.Request) error {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(req.RawBody), &parsed); err != nil {
		return fmt.Errorf("request body is not valid JSON: %w", err)
	}

	if v, _ := parsed["jsonrpc"].(string); v != "2.0" {
		return fmt.Errorf("expected jsonrpc \"2.0\", got %q", v)
	}
	if _, ok := parsed["id"]; !ok {
		return fmt.Errorf("request missing \"id\" field")
	}
	if _, ok := parsed["method"]; !ok {
		return fmt.Errorf("request missing \"method\" field")
	}
	return nil
}

// Helper: check Content-Type header.
func RequireContentType(req mockserver.Request, expected string) error {
	ct := req.Headers.Get("Content-Type")
	if ct == "" {
		return fmt.Errorf("request missing Content-Type header")
	}
	if ct != expected && !containsMediaType(ct, expected) {
		return fmt.Errorf("expected Content-Type %q, got %q", expected, ct)
	}
	return nil
}

func containsMediaType(actual, expected string) bool {
	// Simple substring check for media type matching
	return len(actual) > 0 && len(expected) > 0 &&
		(actual == expected || (len(actual) > len(expected) && actual[:len(expected)] == expected))
}

func resolveJSONPath(data map[string]any, path string) any {
	parts := splitPath(path)
	var current any = data
	for _, part := range parts {
		switch m := current.(type) {
		case map[string]any:
			current = m[part]
		default:
			return nil
		}
	}
	return current
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
