package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aleksclark/spec-torture/internal/schema"
)

// Format represents the output format for reports.
type Format string

const (
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

// Write renders a TestRun in the specified format to the given writer.
func Write(w io.Writer, run *schema.TestRun, format Format) error {
	switch format {
	case FormatJSON:
		return writeJSON(w, run)
	case FormatMarkdown:
		return writeMarkdown(w, run)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func writeJSON(w io.Writer, run *schema.TestRun) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(run)
}

func writeMarkdown(w io.Writer, run *schema.TestRun) error {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Test Run: %s\n\n", run.ID))
	b.WriteString(fmt.Sprintf("**Spec:** %s  \n", run.SpecID))
	b.WriteString(fmt.Sprintf("**Runtime:** %s %s  \n", run.Runtime, run.RuntimeVersion))
	b.WriteString(fmt.Sprintf("**Started:** %s  \n", run.StartedAt.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("**Completed:** %s  \n\n", run.CompletedAt.Format("2006-01-02 15:04:05")))

	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Count |\n"))
	b.WriteString(fmt.Sprintf("|--------|-------|\n"))
	b.WriteString(fmt.Sprintf("| Total | %d |\n", run.Summary.Total))
	b.WriteString(fmt.Sprintf("| Passed | %d |\n", run.Summary.Passed))
	b.WriteString(fmt.Sprintf("| Failed | %d |\n", run.Summary.Failed))
	b.WriteString(fmt.Sprintf("| Errors | %d |\n", run.Summary.Errors))
	b.WriteString(fmt.Sprintf("| Skipped | %d |\n", run.Summary.Skipped))
	b.WriteString(fmt.Sprintf("| Timeouts | %d |\n", run.Summary.Timeouts))
	b.WriteString(fmt.Sprintf("| **Compliance** | **%.1f%%** |\n\n", run.Summary.Compliance))

	b.WriteString("## Results\n\n")
	b.WriteString("| Test Case | Status | Duration | Error |\n")
	b.WriteString("|-----------|--------|----------|-------|\n")

	for _, r := range run.Results {
		statusIcon := statusEmoji(r.Status)
		errMsg := r.ErrorMessage
		if len(errMsg) > 80 {
			errMsg = errMsg[:80] + "..."
		}
		b.WriteString(fmt.Sprintf("| %s | %s %s | %s | %s |\n",
			r.TestCaseID, statusIcon, r.Status, r.Duration, errMsg))
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func statusEmoji(s schema.Status) string {
	switch s {
	case schema.StatusPass:
		return "PASS"
	case schema.StatusFail:
		return "FAIL"
	case schema.StatusError:
		return "ERR "
	case schema.StatusSkip:
		return "SKIP"
	case schema.StatusTimeout:
		return "TIME"
	default:
		return "????"
	}
}
