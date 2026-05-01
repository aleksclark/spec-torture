package schema

import (
	"fmt"
	"time"
)

// Severity maps to RFC 2119 requirement levels.
type Severity string

const (
	SeverityRequired    Severity = "required"
	SeverityRecommended Severity = "recommended"
	SeverityOptional    Severity = "optional"
)

// Action defines what a test step does.
type Action string

const (
	ActionSend        Action = "send"
	ActionExpect      Action = "expect"
	ActionWait        Action = "wait"
	ActionAssert      Action = "assert"
	ActionExec        Action = "exec"
	ActionMCPToolCall Action = "mcp_tool_call"
)

// Step is an individual action within a test case.
type Step struct {
	Action  Action         `yaml:"action" json:"action"`
	Payload map[string]any `yaml:"payload,omitempty" json:"payload,omitempty"`
	Expect  map[string]any `yaml:"expect,omitempty" json:"expect,omitempty"`
	Timeout time.Duration  `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// Setup describes optional Docker/config needed before a test.
type Setup struct {
	Image   string            `yaml:"image,omitempty" json:"image,omitempty"`
	Command []string          `yaml:"command,omitempty" json:"command,omitempty"`
	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

// TestCase is an individual test within a spec.
type TestCase struct {
	ID          string        `yaml:"id" json:"id"`
	Name        string        `yaml:"name" json:"name"`
	Description string        `yaml:"description" json:"description"`
	Severity    Severity      `yaml:"severity" json:"severity"`
	Category    string        `yaml:"category" json:"category"`
	Setup       *Setup        `yaml:"setup,omitempty" json:"setup,omitempty"`
	Steps       []Step        `yaml:"steps" json:"steps"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
	Tags        []string      `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Validate checks that a TestCase has all required fields.
func (tc *TestCase) Validate() []error {
	var errs []error

	if tc.ID == "" {
		errs = append(errs, &ValidationError{Field: "test_case.id", Message: "must not be empty"})
	}
	if tc.Name == "" {
		errs = append(errs, &ValidationError{Field: fmt.Sprintf("test_case[%s].name", tc.ID), Message: "must not be empty"})
	}

	validSeverities := map[Severity]bool{
		SeverityRequired:    true,
		SeverityRecommended: true,
		SeverityOptional:    true,
	}
	if tc.Severity != "" && !validSeverities[tc.Severity] {
		errs = append(errs, &ValidationError{
			Field:   fmt.Sprintf("test_case[%s].severity", tc.ID),
			Message: "must be one of: required, recommended, optional",
		})
	}

	if len(tc.Steps) == 0 {
		errs = append(errs, &ValidationError{
			Field:   fmt.Sprintf("test_case[%s].steps", tc.ID),
			Message: "must have at least one step",
		})
	}

	validActions := map[Action]bool{
		ActionSend:        true,
		ActionExpect:      true,
		ActionWait:        true,
		ActionAssert:      true,
		ActionExec:        true,
		ActionMCPToolCall: true,
	}
	for i, step := range tc.Steps {
		if !validActions[step.Action] {
			errs = append(errs, &ValidationError{
				Field:   fmt.Sprintf("test_case[%s].steps[%d].action", tc.ID, i),
				Message: "must be one of: send, expect, wait, assert, exec, mcp_tool_call",
			})
		}
	}

	return errs
}

// ValidationError represents a schema validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
}
