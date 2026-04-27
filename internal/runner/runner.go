package runner

import (
	"context"
	"fmt"
	"log/slog"
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
	Tags           []string
	Timeout        time.Duration
}

// Runner executes specs against a runtime.
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
		if spec.Transport == schema.TransportHTTPREST && cfg.BaseURL != "" {
			result = r.runHTTPTestCase(ctx, spec, &tc, cfg)
		} else {
			result = r.runDockerTestCase(ctx, spec, &tc, cfg)
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

// runHTTPTestCase runs a test case against an external HTTP server.
func (r *Runner) runHTTPTestCase(ctx context.Context, spec *schema.Spec, tc *schema.TestCase, cfg Config) schema.TestResult {
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

	ht := NewHTTPTransport(cfg.BaseURL, r.eval, r.logger)

	for i, step := range tc.Steps {
		stepResult := ht.ExecuteStep(testCtx, &step, i)
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

// runDockerTestCase runs a test case in a Docker container (original behavior).
func (r *Runner) runDockerTestCase(ctx context.Context, spec *schema.Spec, tc *schema.TestCase, cfg Config) schema.TestResult {
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
		stepResult := r.executeStep(testCtx, containerID, spec.Transport, &step, i)
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

func (r *Runner) executeStep(ctx context.Context, containerID string, transport schema.Transport, step *schema.Step, index int) schema.StepResult {
	start := time.Now()
	sr := schema.StepResult{
		StepIndex: index,
		Action:    step.Action,
	}

	r.logger.Debug("executing step", "index", index, "action", step.Action)

	switch step.Action {
	case schema.ActionSend:
		sr.Status = schema.StatusError
		sr.Error = "step execution not yet implemented"
	case schema.ActionExpect:
		sr.Status = schema.StatusError
		sr.Error = "step execution not yet implemented"
	case schema.ActionWait:
		sr.Status = schema.StatusError
		sr.Error = "step execution not yet implemented"
	case schema.ActionAssert:
		sr.Status = schema.StatusError
		sr.Error = "step execution not yet implemented"
	case schema.ActionExec:
		sr.Status = schema.StatusError
		sr.Error = "step execution not yet implemented"
	default:
		sr.Status = schema.StatusError
		sr.Error = fmt.Sprintf("unknown action: %s", step.Action)
	}

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
