package reporter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"github.com/timkrebs/custos/pkg/evaluator"
)

// Terminal renders test results as colored terminal output.
// The output format matches the custos README specification:
//
//	payment-service-policies
//
//	  OK can read its own secrets          (secret/data/payment-svc/db-creds)
//	  FAIL no access to sys backend        (sys/seal)
//	    → expected: deny, got: allow via policy "admin-legacy"
//
//	4 passed · 1 failed · 0 skipped
type Terminal struct {
	Writer  io.Writer
	Verbose bool
}

// NewTerminal creates a terminal reporter writing to the given writer.
// Pass nil to write to os.Stdout.
func NewTerminal(w io.Writer, verbose bool) *Terminal {
	if w == nil {
		w = os.Stdout
	}
	return &Terminal{Writer: w, Verbose: verbose}
}

// Report renders a full suite result to the terminal.
func (t *Terminal) Report(suite evaluator.SuiteResult) {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)
	cyan := color.New(color.FgCyan)
	bold := color.New(color.Bold)
	dim := color.New(color.Faint)

	// Suite header.
	fmt.Fprintln(t.Writer)
	bold.Fprintf(t.Writer, "  %s\n", suite.Suite)
	fmt.Fprintln(t.Writer)

	// Individual test results.
	for _, tr := range suite.Results {
		path := tr.Test.Path
		if tr.Pass {
			green.Fprintf(t.Writer, "    OK ")
			fmt.Fprintf(t.Writer, "%-45s ", tr.Test.Name)
			dim.Fprintf(t.Writer, "(%s)\n", path)
		} else {
			red.Fprintf(t.Writer, "    FAIL ")
			fmt.Fprintf(t.Writer, "%-43s ", tr.Test.Name)
			dim.Fprintf(t.Writer, "(%s)\n", path)

			// Failure detail line.
			expected := tr.Test.Expect
			got := "deny"
			if tr.Result.Allowed {
				got = "allow"
			}
			detail := fmt.Sprintf("expected: %s, got: %s", expected, got)
			if tr.Result.MatchedRule != nil {
				policyName := filepath.Base(tr.Result.MatchedRule.PolicyFile)
				detail += fmt.Sprintf(" via policy %q", strings.TrimSuffix(policyName, ".hcl"))
			}
			yellow.Fprintf(t.Writer, "      → %s\n", detail)
		}

		// Verbose trace.
		if t.Verbose {
			cyan.Fprintf(t.Writer, "      %s\n", tr.Result.Explanation)
		}
	}

	// Summary line.
	fmt.Fprintln(t.Writer)
	skipped := 0
	passStr := fmt.Sprintf("%d passed", suite.Passed)
	failStr := fmt.Sprintf("%d failed", suite.Failed)
	skipStr := fmt.Sprintf("%d skipped", skipped)

	fmt.Fprint(t.Writer, "  ")
	if suite.Passed > 0 {
		green.Fprint(t.Writer, passStr)
	} else {
		fmt.Fprint(t.Writer, passStr)
	}
	dim.Fprint(t.Writer, " · ")
	if suite.Failed > 0 {
		red.Fprint(t.Writer, failStr)
	} else {
		fmt.Fprint(t.Writer, failStr)
	}
	dim.Fprint(t.Writer, " · ")
	fmt.Fprintln(t.Writer, skipStr)
	fmt.Fprintln(t.Writer)
}
