package runner

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"reflect"
)

// Evaluator checks actual responses against expected patterns.
type Evaluator struct {
	logger *slog.Logger
}

// NewEvaluator creates a new Evaluator.
func NewEvaluator(logger *slog.Logger) *Evaluator {
	return &Evaluator{logger: logger}
}

// Match checks whether the actual response matches the expected pattern.
// The expected map supports partial matching: only the keys present in expected
// need to appear in actual. Nested maps are matched recursively.
func (e *Evaluator) Match(expected, actual map[string]any) error {
	return matchMaps("", expected, actual)
}

func matchMaps(path string, expected, actual map[string]any) error {
	for key, expVal := range expected {
		p := key
		if path != "" {
			p = path + "." + key
		}

		actVal, ok := actual[key]
		if !ok {
			return fmt.Errorf("missing key at %s", p)
		}

		if err := matchValues(p, expVal, actVal); err != nil {
			return err
		}
	}
	return nil
}

func matchValues(path string, expected, actual any) error {
	switch exp := expected.(type) {
	case map[string]any:
		act, ok := actual.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object at %s, got %T", path, actual)
		}
		return matchMaps(path, exp, act)

	case []any:
		act, ok := actual.([]any)
		if !ok {
			return fmt.Errorf("expected array at %s, got %T", path, actual)
		}
		if len(act) < len(exp) {
			return fmt.Errorf("expected at least %d items at %s, got %d", len(exp), path, len(act))
		}
		for i, v := range exp {
			if err := matchValues(fmt.Sprintf("%s[%d]", path, i), v, act[i]); err != nil {
				return err
			}
		}

	case string:
		if exp == "*" {
			return nil
		}
		actStr := fmt.Sprintf("%v", actual)
		if exp != actStr {
			return fmt.Errorf("mismatch at %s: expected %q, got %q", path, exp, actStr)
		}

	case json.Number:
		if !numericEqual(expected, actual) {
			return fmt.Errorf("mismatch at %s: expected %v, got %v", path, expected, actual)
		}

	default:
		if !reflect.DeepEqual(expected, actual) {
			if numericEqual(expected, actual) {
				return nil
			}
			return fmt.Errorf("mismatch at %s: expected %v (%T), got %v (%T)",
				path, expected, expected, actual, actual)
		}
	}

	return nil
}

// numericEqual compares two values numerically, handling int/float64/json.Number mismatches.
func numericEqual(a, b any) bool {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if aOk && bOk {
		return math.Abs(af-bf) < 1e-9
	}
	return false
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
