package conformance

import (
	"context"
	"testing"
)

// TestConformance boots the reference server with the mock backend and runs the
// full ARP conformance suite, failing on any required-severity violation.
func TestConformance(t *testing.T) {
	env, err := Start(context.Background())
	if err != nil {
		t.Fatalf("start env: %v", err)
	}
	defer env.Stop()

	results, err := Run(env)
	if err != nil {
		t.Fatalf("run suite: %v", err)
	}
	for _, r := range results {
		if !r.Pass && r.Severity == Required {
			t.Errorf("REQUIRED FAIL %s/%s: %s", r.Group, r.ID, r.Detail)
		}
	}
	s := Summarize(results)
	t.Logf("ARP conformance: %d/%d passed (%.1f%%), %d required failures",
		s.Passed, s.Total, s.Compliance, len(RequiredFailures(results)))
}
