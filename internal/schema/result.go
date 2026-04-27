package schema

import "time"

// Status represents the outcome of a test case execution.
type Status string

const (
	StatusPass    Status = "pass"
	StatusFail    Status = "fail"
	StatusError   Status = "error"
	StatusSkip    Status = "skip"
	StatusTimeout Status = "timeout"
)

// StepResult captures the outcome of a single step execution.
type StepResult struct {
	StepIndex int           `json:"step_index"`
	Action    Action        `json:"action"`
	Status    Status        `json:"status"`
	Duration  time.Duration `json:"duration"`
	Expected  any           `json:"expected,omitempty"`
	Actual    any           `json:"actual,omitempty"`
	Error     string        `json:"error,omitempty"`
}

// TestResult is the outcome of running a single test case against a runtime.
type TestResult struct {
	ID             string       `json:"id"`
	SpecID         string       `json:"spec_id"`
	TestCaseID     string       `json:"test_case_id"`
	Runtime        string       `json:"runtime"`
	RuntimeVersion string       `json:"runtime_version"`
	Status         Status       `json:"status"`
	Duration       time.Duration `json:"duration"`
	StartedAt      time.Time    `json:"started_at"`
	CompletedAt    time.Time    `json:"completed_at"`
	Steps          []StepResult `json:"steps"`
	ErrorMessage   string       `json:"error_message,omitempty"`
	RawLog         string       `json:"raw_log,omitempty"`
}

// Summary aggregates pass/fail/error/skip counts for a test run.
type Summary struct {
	Total      int     `json:"total"`
	Passed     int     `json:"passed"`
	Failed     int     `json:"failed"`
	Errors     int     `json:"errors"`
	Skipped    int     `json:"skipped"`
	Timeouts   int     `json:"timeouts"`
	Compliance float64 `json:"compliance"`
}

// TestRun represents a full run of a spec (or subset) against a runtime.
type TestRun struct {
	ID             string       `json:"id"`
	SpecID         string       `json:"spec_id"`
	Runtime        string       `json:"runtime"`
	RuntimeVersion string       `json:"runtime_version"`
	StartedAt      time.Time    `json:"started_at"`
	CompletedAt    time.Time    `json:"completed_at"`
	Results        []TestResult `json:"results"`
	Summary        Summary      `json:"summary"`
}

// ComputeSummary recalculates the Summary from the Results slice.
func (tr *TestRun) ComputeSummary() {
	s := Summary{Total: len(tr.Results)}
	for _, r := range tr.Results {
		switch r.Status {
		case StatusPass:
			s.Passed++
		case StatusFail:
			s.Failed++
		case StatusError:
			s.Errors++
		case StatusSkip:
			s.Skipped++
		case StatusTimeout:
			s.Timeouts++
		}
	}
	applicable := s.Total - s.Skipped
	if applicable > 0 {
		s.Compliance = float64(s.Passed) / float64(applicable) * 100
	}
	tr.Summary = s
}
