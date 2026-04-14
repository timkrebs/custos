package reporter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
		contribsShown := false
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

			// On failure, surface multi-policy provenance so users can see
			// which policy denied and which policies would have granted.
			if hasMultiPolicyProvenance(tr.Result.Composed) {
				t.renderContributions(tr.Result.Composed)
				contribsShown = true
			}
		}

		// Verbose trace.
		if t.Verbose {
			cyan.Fprintf(t.Writer, "      %s\n", tr.Result.Explanation)
			if !contribsShown && hasMultiPolicyProvenance(tr.Result.Composed) {
				t.renderContributions(tr.Result.Composed)
			}
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

// hasMultiPolicyProvenance reports whether a composed result has enough
// cross-policy provenance to be worth rendering. Single-policy results are
// already covered by the "via policy" line in the failure detail, so we
// only emit the contributions block when at least two policies contributed.
func hasMultiPolicyProvenance(c *evaluator.Composed) bool {
	return c != nil && len(c.Contributions) >= 2
}

// renderContributions writes a compact block listing each policy that
// contributed to the composed decision, the rule path that matched within
// that policy, and either the capabilities it granted or a DENIED marker.
// The block is indented to align under the failure/verbose detail lines.
func (t *Terminal) renderContributions(c *evaluator.Composed) {
	if c == nil || len(c.Contributions) == 0 {
		return
	}
	dim := color.New(color.Faint)
	red := color.New(color.FgRed)

	dim.Fprintln(t.Writer, "      contributions:")
	for _, contrib := range c.Contributions {
		name := strings.TrimSuffix(filepath.Base(contrib.PolicyFile), ".hcl")
		if contrib.IsDeny {
			red.Fprintf(t.Writer, "        - %s (%s) DENIED\n", name, contrib.RulePath)
			continue
		}
		grants := nonDenyCapabilities(contrib.Capabilities)
		dim.Fprintf(t.Writer, "        - %s (%s) granted %v\n", name, contrib.RulePath, grants)
	}
}

// nonDenyCapabilities returns the capability list with any "deny" sentinel
// stripped, sorted for deterministic output. Used by the contributions
// renderer so downstream test assertions and golden snapshots are stable.
func nonDenyCapabilities(caps []string) []string {
	out := make([]string, 0, len(caps))
	for _, c := range caps {
		if c == "deny" {
			continue
		}
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}
