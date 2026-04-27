package runner

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aleksclark/spec-torture/internal/schema"
	"github.com/google/uuid"
)

// Config holds runtime configuration for a test run.
type Config struct {
	Runtime        string
	RuntimeVersion string
	Image          string
	BaseURL        string
	RPCPath        string
	Tags           []string
	Timeout        time.Duration
}

// Runner executes specs against a runtime in Docker containers or against a URL.
type Runner struct {
	docker *DockerManager
	eval   *Evaluator
	logger *slog.Logger
	config Config
}

// New creates a new Runner instance. Docker is only initialized if needed.
func New(logger *slog.Logger, cfg Config) (*Runner, error) {
	r := &Runner{
		eval:   NewEvaluator(logger),
		logger: logger,
		config: cfg,
	}

	// Only initialize Docker if we don't have a base URL (i.e., not http-rest against external server)
	if cfg.BaseURL == "" {
		dm, err := NewDockerManager(logger)
		if err != nil {
			return nil, fmt.Errorf("creating docker manager: %w", err)
		}
		r.docker = dm
	}

	return r, nil
}

// Run executes all test cases in a spec against the configured runtime.
func (r *Runner) Run(ctx context.Context, spec *schema.Spec, cfg Config) (*schema.TestRun, error) {
	run := &schema.TestRun{
		ID:             uuid.New().String(),
		SpecID:         spec.ID,
		Runtime:        cfg.Runtime,
		RuntimeVersion: cfg.RuntimeVersion,
		StartedAt:      time.Now(),
	}

	r.logger.Info("starting test run",
		"run_id", run.ID,
		"spec", spec.ID,
		"runtime", cfg.Runtime,
		"transport", spec.Transport,
	)

	for _, tc := range spec.TestCases {
		if len(cfg.Tags) > 0 && !matchesTags(tc.Tags, cfg.Tags) {
			run.Results = append(run.Results, schema.TestResult{
				ID:         uuid.New().String(),
				SpecID:     spec.ID,
				TestCaseID: tc.ID,
				Runtime:    cfg.Runtime,
				Status:     schema.StatusSkip,
			})
			continue
		}

		var result schema.TestResult
		if cfg.BaseURL != "" {
			result = r.runTestCaseHTTP(ctx, spec, &tc, cfg)
		} else {
			result = r.runTestCaseDocker(ctx, spec, &tc, cfg)
		}
		run.Results = append(run.Results, result)
	}

	run.CompletedAt = time.Now()
	run.ComputeSummary()

	r.logger.Info("test run complete",
		"run_id", run.ID,
		"passed", run.Summary.Passed,
		"failed", run.Summary.Failed,
		"errors", run.Summary.Errors,
		"compliance", fmt.Sprintf("%.1f%%", run.Summary.Compliance),
	)

	return run, nil
}

// runTestCaseHTTP runs a test case against a live HTTP endpoint.
func (r *Runner) runTestCaseHTTP(ctx context.Context, spec *schema.Spec, tc *schema.TestCase, cfg Config) schema.TestResult {
	result := schema.TestResult{
		ID:             uuid.New().String(),
		SpecID:         spec.ID,
		TestCaseID:     tc.ID,
		Runtime:        cfg.Runtime,
		RuntimeVersion: cfg.RuntimeVersion,
		StartedAt:      time.Now(),
	}

	timeout := tc.Timeout
	if timeout == 0 {
		timeout = cfg.Timeout
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	r.logger.Info("running test case", "test_case", tc.ID, "category", tc.Category)

	transport := r.createTransport(spec.Transport, cfg.BaseURL, cfg.RPCPath)
	if transport == nil {
		result.Status = schema.StatusError
		result.ErrorMessage = fmt.Sprintf("unsupported transport: %s", spec.Transport)
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		return result
	}

	for i, step := range tc.Steps {
		stepResult := r.executeStepWithTransport(testCtx, transport, &step, i)
		result.Steps = append(result.Steps, stepResult)

		if stepResult.Status != schema.StatusPass {
			result.Status = stepResult.Status
			result.ErrorMessage = stepResult.Error
			break
		}
	}

	if result.Status == "" {
		result.Status = schema.StatusPass
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	return result
}

// runTestCaseDocker runs a test case using Docker (original behavior).
func (r *Runner) runTestCaseDocker(ctx context.Context, spec *schema.Spec, tc *schema.TestCase, cfg Config) schema.TestResult {
	result := schema.TestResult{
		ID:             uuid.New().String(),
		SpecID:         spec.ID,
		TestCaseID:     tc.ID,
		Runtime:        cfg.Runtime,
		RuntimeVersion: cfg.RuntimeVersion,
		StartedAt:      time.Now(),
	}

	timeout := tc.Timeout
	if timeout == 0 {
		timeout = cfg.Timeout
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	r.logger.Info("running test case", "test_case", tc.ID, "category", tc.Category)

	if r.docker == nil {
		result.Status = schema.StatusError
		result.ErrorMessage = "Docker not available and no base URL configured"
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		return result
	}

	image := cfg.Image
	if tc.Setup != nil && tc.Setup.Image != "" {
		image = tc.Setup.Image
	}

	containerID, err := r.docker.StartContainer(testCtx, image, tc.Setup)
	if err != nil {
		result.Status = schema.StatusError
		result.ErrorMessage = fmt.Sprintf("failed to start container: %v", err)
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		return result
	}
	defer func() {
		if err := r.docker.StopContainer(context.Background(), containerID); err != nil {
			r.logger.Warn("failed to stop container", "container", containerID, "error", err)
		}
	}()

	for i, step := range tc.Steps {
		stepResult := r.executeStepLegacy(testCtx, containerID, spec.Transport, &step, i)
		result.Steps = append(result.Steps, stepResult)

		if stepResult.Status != schema.StatusPass {
			result.Status = stepResult.Status
			result.ErrorMessage = stepResult.Error
			break
		}
	}

	if result.Status == "" {
		result.Status = schema.StatusPass
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	return result
}

func (r *Runner) createTransport(transport schema.Transport, baseURL, rpcPath string) Transport {
	switch transport {
	case schema.TransportJSONRPCHTTP:
		return NewJSONRPCTransport(baseURL, rpcPath, r.logger)
	case schema.TransportHTTPREST:
		return NewHTTPTransport(baseURL, r.logger)
	default:
		return nil
	}
}

// executeStepWithTransport executes a step using the transport interface.
func (r *Runner) executeStepWithTransport(ctx context.Context, transport Transport, step *schema.Step, index int) schema.StepResult {
	start := time.Now()
	sr := schema.StepResult{
		StepIndex: index,
		Action:    step.Action,
	}

	r.logger.Debug("executing step", "index", index, "action", step.Action)

	switch step.Action {
	case schema.ActionSend:
		resp, err := transport.Send(ctx, step)
		if err != nil {
			sr.Status = schema.StatusError
			sr.Error = fmt.Sprintf("send failed: %v", err)
		} else {
			sr.Status = schema.StatusPass
			sr.Actual = resp
		}

	case schema.ActionExpect:
		lastResp := transport.LastResponse()
		if lastResp == nil {
			sr.Status = schema.StatusError
			sr.Error = "no previous response to check expectations against"
		} else {
			sr.Expected = step.Expect
			sr.Actual = lastResp
			if err := matchExpect(r.eval, lastResp, step.Expect); err != nil {
				sr.Status = schema.StatusFail
				sr.Error = err.Error()
			} else {
				sr.Status = schema.StatusPass
			}
		}

	case schema.ActionWait:
		duration := 1 * time.Second
		if d, ok := step.Payload["duration"]; ok {
			if ds, ok := d.(string); ok {
				if parsed, err := time.ParseDuration(ds); err == nil {
					duration = parsed
				}
			}
		}
		select {
		case <-time.After(duration):
			sr.Status = schema.StatusPass
		case <-ctx.Done():
			sr.Status = schema.StatusTimeout
			sr.Error = "wait timed out"
		}

	case schema.ActionAssert:
		lastResp := transport.LastResponse()
		if lastResp == nil {
			sr.Status = schema.StatusError
			sr.Error = "no previous response to assert against"
		} else {
			if err := r.evaluateAssert(step.Expect, lastResp); err != nil {
				sr.Status = schema.StatusFail
				sr.Error = err.Error()
			} else {
				sr.Status = schema.StatusPass
			}
		}

	case schema.ActionExec:
		sr.Status = schema.StatusSkip
		sr.Error = "exec action not supported in HTTP mode"

	default:
		sr.Status = schema.StatusError
		sr.Error = fmt.Sprintf("unknown action: %s", step.Action)
	}

	sr.Duration = time.Since(start)
	return sr
}

// evaluateAssert handles assert steps with condition expressions.
func (r *Runner) evaluateAssert(expect map[string]any, resp *TransportResponse) error {
	cond, ok := expect["condition"]
	if !ok {
		return fmt.Errorf("assert step missing 'condition'")
	}

	condStr, ok := cond.(string)
	if !ok {
		return fmt.Errorf("assert condition must be a string")
	}

	data := mergeResponseData(resp)
	if data == nil {
		return fmt.Errorf("no response data to assert against")
	}

	// Parse simple "field in [values]" conditions
	if strings.Contains(condStr, " in [") {
		return r.evaluateInCondition(condStr, data)
	}

	return fmt.Errorf("unsupported assert condition format: %s", condStr)
}

// evaluateInCondition evaluates "field.path in ['val1', 'val2']" conditions.
func (r *Runner) evaluateInCondition(cond string, data map[string]any) error {
	parts := strings.SplitN(cond, " in ", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid 'in' condition: %s", cond)
	}

	fieldPath := strings.TrimSpace(parts[0])
	valuesStr := strings.TrimSpace(parts[1])

	// Resolve the field value
	actual := resolveJSONPath(data, fieldPath)
	if actual == nil {
		return fmt.Errorf("field %q not found in response", fieldPath)
	}
	actualStr := fmt.Sprintf("%v", actual)

	// Parse the values list
	valuesStr = strings.Trim(valuesStr, "[]")
	values := strings.Split(valuesStr, ",")
	for _, v := range values {
		v = strings.TrimSpace(v)
		v = strings.Trim(v, "'\"")
		if v == actualStr {
			return nil
		}
	}

	return fmt.Errorf("value %q not in allowed set %s", actualStr, parts[1])
}

// executeStepLegacy is the old stub for Docker-based execution.
func (r *Runner) executeStepLegacy(_ context.Context, _ string, _ schema.Transport, step *schema.Step, index int) schema.StepResult {
	start := time.Now()
	sr := schema.StepResult{
		StepIndex: index,
		Action:    step.Action,
	}

	r.logger.Debug("executing step (legacy)", "index", index, "action", step.Action)

	sr.Status = schema.StatusError
	sr.Error = "step execution not yet implemented for Docker transport"

	sr.Duration = time.Since(start)
	return sr
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

// Close releases resources held by the runner.
func (r *Runner) Close() error {
	if r.docker != nil {
		return r.docker.Close()
	}
	return nil
}
