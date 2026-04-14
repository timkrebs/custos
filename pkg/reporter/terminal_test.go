package reporter

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"

	"github.com/timkrebs/custos/pkg/evaluator"
	"github.com/timkrebs/custos/pkg/spec"
)

func init() {
	// Disable colors in tests so output assertions are predictable.
	color.NoColor = true
}

func TestTerminal_Report_AllPass(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, false)

	suite := evaluator.SuiteResult{
		Suite:  "payment-service-policies",
		Passed: 2,
		Failed: 0,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "can read secrets", Path: "secret/data/app/db", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{TestName: "can read secrets", Path: "secret/data/app/db", Allowed: true, Explanation: "allowed by rule", MatchedRule: &evaluator.MatchedRule{PolicyFile: "app.hcl", RulePath: "secret/data/app/*"}},
				Pass:   true,
			},
			{
				Test:   spec.TestCase{Name: "deny billing", Path: "secret/data/billing/key", Capabilities: []string{"read"}, Expect: "deny"},
				Result: evaluator.Result{TestName: "deny billing", Path: "secret/data/billing/key", Allowed: false, Explanation: "explicitly denied"},
				Pass:   true,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	// Suite header present.
	if !strings.Contains(out, "payment-service-policies") {
		t.Errorf("output should contain suite name, got:\n%s", out)
	}

	// Both tests show OK.
	if strings.Count(out, "OK ") != 2 {
		t.Errorf("expected 2 OK lines, got:\n%s", out)
	}

	// No FAIL lines.
	if strings.Contains(out, "FAIL ") {
		t.Errorf("should not contain FAIL, got:\n%s", out)
	}

	// Summary line.
	if !strings.Contains(out, "2 passed") {
		t.Errorf("summary should show 2 passed, got:\n%s", out)
	}
	if !strings.Contains(out, "0 failed") {
		t.Errorf("summary should show 0 failed, got:\n%s", out)
	}
	if !strings.Contains(out, "0 skipped") {
		t.Errorf("summary should show 0 skipped, got:\n%s", out)
	}
}

func TestTerminal_Report_WithFailure(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, false)

	suite := evaluator.SuiteResult{
		Suite:  "test-suite",
		Passed: 1,
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "allow read", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{TestName: "allow read", Path: "secret/foo", Allowed: true, Explanation: "allowed"},
				Pass:   true,
			},
			{
				Test:   spec.TestCase{Name: "no sys access", Path: "sys/seal", Capabilities: []string{"sudo"}, Expect: "deny"},
				Result: evaluator.Result{TestName: "no sys access", Path: "sys/seal", Allowed: true, Explanation: "allowed by admin", MatchedRule: &evaluator.MatchedRule{PolicyFile: "admin-legacy.hcl", RulePath: "sys/*"}},
				Pass:   false,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	// FAIL line present.
	if !strings.Contains(out, "FAIL ") {
		t.Errorf("should contain FAIL, got:\n%s", out)
	}

	// Arrow detail line with expected/got.
	if !strings.Contains(out, "→") {
		t.Errorf("should contain arrow detail, got:\n%s", out)
	}
	if !strings.Contains(out, "expected: deny, got: allow") {
		t.Errorf("should show expected vs got, got:\n%s", out)
	}

	// Policy name in detail (without .hcl extension).
	if !strings.Contains(out, `via policy "admin-legacy"`) {
		t.Errorf("should show policy name, got:\n%s", out)
	}

	// Summary.
	if !strings.Contains(out, "1 passed") || !strings.Contains(out, "1 failed") {
		t.Errorf("summary wrong, got:\n%s", out)
	}
}

func TestTerminal_Report_Verbose(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, true)

	suite := evaluator.SuiteResult{
		Suite:  "verbose-suite",
		Passed: 1,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "allow read", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{TestName: "allow read", Path: "secret/foo", Allowed: true, Explanation: "allowed by rule \"secret/*\" in app.hcl"},
				Pass:   true,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	// Verbose mode shows the explanation trace.
	if !strings.Contains(out, `allowed by rule "secret/*" in app.hcl`) {
		t.Errorf("verbose mode should show explanation, got:\n%s", out)
	}
}

func TestTerminal_Report_ImplicitDeny(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, false)

	suite := evaluator.SuiteResult{
		Suite:  "deny-suite",
		Passed: 1,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "no match", Path: "secret/other", Capabilities: []string{"read"}, Expect: "deny"},
				Result: evaluator.Result{TestName: "no match", Path: "secret/other", Allowed: false, MatchedRule: nil, Explanation: "no policy rule matches path (implicit deny)"},
				Pass:   true,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	// Should show OK, not contain policy detail.
	if !strings.Contains(out, "OK ") {
		t.Errorf("implicit deny matching expect should be OK, got:\n%s", out)
	}
}

func TestTerminal_Report_FailureWithoutMatchedRule(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, false)

	suite := evaluator.SuiteResult{
		Suite:  "fail-no-rule",
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "should allow", Path: "secret/missing", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{TestName: "should allow", Path: "secret/missing", Allowed: false, MatchedRule: nil, Explanation: "no policy rule matches path (implicit deny)"},
				Pass:   false,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	// FAIL present, arrow line present.
	if !strings.Contains(out, "FAIL ") {
		t.Errorf("should contain FAIL, got:\n%s", out)
	}
	if !strings.Contains(out, "expected: allow, got: deny") {
		t.Errorf("should show expected vs got, got:\n%s", out)
	}
	// Should NOT contain "via policy" since MatchedRule is nil.
	if strings.Contains(out, "via policy") {
		t.Errorf("should not reference policy when MatchedRule is nil, got:\n%s", out)
	}
}

func TestTerminal_Report_EmptySuite(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, false)

	suite := evaluator.SuiteResult{Suite: "empty"}
	r.Report(suite)
	out := buf.String()

	if !strings.Contains(out, "empty") {
		t.Errorf("should show suite name, got:\n%s", out)
	}
	if !strings.Contains(out, "0 passed") {
		t.Errorf("should show 0 passed, got:\n%s", out)
	}
}

func TestTerminal_NilWriter_DefaultsToStdout(t *testing.T) {
	r := NewTerminal(nil, false)
	if r.Writer != os.Stdout {
		t.Error("nil writer should default to os.Stdout")
	}
}

func TestTerminal_Report_PathsInOutput(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, false)

	suite := evaluator.SuiteResult{
		Suite:  "path-suite",
		Passed: 1,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "check path", Path: "secret/data/payment-svc/db-creds", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{TestName: "check path", Path: "secret/data/payment-svc/db-creds", Allowed: true, Explanation: "allowed"},
				Pass:   true,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	// Path should appear in parentheses.
	if !strings.Contains(out, "(secret/data/payment-svc/db-creds)") {
		t.Errorf("output should contain path in parens, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// Multi-policy provenance rendering tests
// ---------------------------------------------------------------------------

// TestTerminal_Report_MultiPolicyFailure_RendersContributions verifies that
// a failing result with multiple contributing policies emits the
// "contributions:" block listing each policy, its matching rule path, and
// the capabilities it granted. This is the primary provenance case: the
// user sees exactly which policy denied and which policies would have
// otherwise granted access.
func TestTerminal_Report_MultiPolicyFailure_RendersContributions(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, false)

	composed := &evaluator.Composed{
		Path:    "secret/data/billing-svc/api-key",
		Granted: map[string]bool{"read": true, "list": true},
		Denied:  true,
		Contributions: []evaluator.RuleContribution{
			{PolicyFile: "policies/readonly.hcl", RulePath: "secret/*", Capabilities: []string{"read", "list"}},
			{PolicyFile: "policies/payment-svc.hcl", RulePath: "secret/data/billing-svc/*", Capabilities: []string{"deny"}, IsDeny: true},
		},
		DeniedBy: []evaluator.RuleContribution{
			{PolicyFile: "policies/payment-svc.hcl", RulePath: "secret/data/billing-svc/*", Capabilities: []string{"deny"}, IsDeny: true},
		},
	}

	suite := evaluator.SuiteResult{
		Suite:  "billing-deny",
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test: spec.TestCase{Name: "should allow read", Path: "secret/data/billing-svc/api-key", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{
					TestName:    "should allow read",
					Path:        "secret/data/billing-svc/api-key",
					Allowed:     false,
					Explanation: "explicitly denied by rule \"secret/data/billing-svc/*\" in policies/payment-svc.hcl",
					MatchedRule: &evaluator.MatchedRule{PolicyFile: "policies/payment-svc.hcl", RulePath: "secret/data/billing-svc/*"},
					Composed:    composed,
				},
				Pass: false,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	if !strings.Contains(out, "contributions:") {
		t.Errorf("missing contributions block, got:\n%s", out)
	}
	if !strings.Contains(out, "readonly (secret/*) granted [list read]") {
		t.Errorf("missing readonly grant line, got:\n%s", out)
	}
	if !strings.Contains(out, "payment-svc (secret/data/billing-svc/*) DENIED") {
		t.Errorf("missing payment-svc DENIED line, got:\n%s", out)
	}
}

// TestTerminal_Report_MultiPolicyFailure_MissingCaps renders the
// contributions block for a missing-capabilities failure (no explicit
// deny), proving that provenance surfaces even when the failure mode is
// union-insufficient rather than deny-override.
func TestTerminal_Report_MultiPolicyFailure_MissingCaps(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, false)

	composed := &evaluator.Composed{
		Path:    "secret/data/app/key",
		Granted: map[string]bool{"read": true, "list": true},
		Contributions: []evaluator.RuleContribution{
			{PolicyFile: "policies/readonly.hcl", RulePath: "secret/*", Capabilities: []string{"read", "list"}},
			{PolicyFile: "policies/app.hcl", RulePath: "secret/data/app/*", Capabilities: []string{"read"}},
		},
	}

	suite := evaluator.SuiteResult{
		Suite:  "missing-caps",
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test: spec.TestCase{Name: "should allow create", Path: "secret/data/app/key", Capabilities: []string{"create"}, Expect: "allow"},
				Result: evaluator.Result{
					TestName:    "should allow create",
					Path:        "secret/data/app/key",
					Allowed:     false,
					Explanation: "missing capabilities [create] (granted: [list read])",
					MatchedRule: &evaluator.MatchedRule{PolicyFile: "policies/readonly.hcl", RulePath: "secret/*"},
					Composed:    composed,
				},
				Pass: false,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	if !strings.Contains(out, "contributions:") {
		t.Errorf("missing contributions block, got:\n%s", out)
	}
	if !strings.Contains(out, "readonly (secret/*) granted [list read]") {
		t.Errorf("missing readonly grant line, got:\n%s", out)
	}
	if !strings.Contains(out, "app (secret/data/app/*) granted [read]") {
		t.Errorf("missing app grant line, got:\n%s", out)
	}
	// No policy denied, so the DENIED marker must not appear.
	if strings.Contains(out, "DENIED") {
		t.Errorf("should not render DENIED marker without a deny contribution, got:\n%s", out)
	}
}

// TestTerminal_Report_SinglePolicyFailure_SuppressesContributions ensures
// the contributions block stays quiet when only one policy contributed.
// Single-policy provenance is already conveyed by the existing "via policy"
// line in the failure detail; repeating it would be noise.
func TestTerminal_Report_SinglePolicyFailure_SuppressesContributions(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, false)

	composed := &evaluator.Composed{
		Path:    "secret/foo",
		Granted: map[string]bool{"read": true},
		Contributions: []evaluator.RuleContribution{
			{PolicyFile: "policies/only.hcl", RulePath: "secret/foo", Capabilities: []string{"read"}},
		},
	}

	suite := evaluator.SuiteResult{
		Suite:  "single-policy",
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test: spec.TestCase{Name: "wants create", Path: "secret/foo", Capabilities: []string{"create"}, Expect: "allow"},
				Result: evaluator.Result{
					TestName:    "wants create",
					Path:        "secret/foo",
					Allowed:     false,
					Explanation: "missing capabilities [create]",
					MatchedRule: &evaluator.MatchedRule{PolicyFile: "policies/only.hcl", RulePath: "secret/foo"},
					Composed:    composed,
				},
				Pass: false,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	if strings.Contains(out, "contributions:") {
		t.Errorf("single-contribution result should not render contributions block, got:\n%s", out)
	}
	// But the existing via-policy line must still be present.
	if !strings.Contains(out, `via policy "only"`) {
		t.Errorf("should still show via policy line, got:\n%s", out)
	}
}

// TestTerminal_Report_VerbosePassMultiPolicy shows that verbose mode renders
// the contributions block even on passing tests when multiple policies
// contributed, so operators can opt into full composition traces.
func TestTerminal_Report_VerbosePassMultiPolicy(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, true)

	composed := &evaluator.Composed{
		Path:    "secret/data/payment-svc/db-creds",
		Granted: map[string]bool{"read": true, "list": true},
		Contributions: []evaluator.RuleContribution{
			{PolicyFile: "policies/readonly.hcl", RulePath: "secret/*", Capabilities: []string{"read", "list"}},
			{PolicyFile: "policies/payment-svc.hcl", RulePath: "secret/data/payment-svc/*", Capabilities: []string{"read", "list"}},
		},
	}

	suite := evaluator.SuiteResult{
		Suite:  "verbose-compose",
		Passed: 1,
		Results: []evaluator.TestResult{
			{
				Test: spec.TestCase{Name: "read payment secrets", Path: "secret/data/payment-svc/db-creds", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{
					TestName:    "read payment secrets",
					Path:        "secret/data/payment-svc/db-creds",
					Allowed:     true,
					Explanation: "allowed by rule \"secret/data/payment-svc/*\" in policies/payment-svc.hcl (composed from 2 policies)",
					MatchedRule: &evaluator.MatchedRule{PolicyFile: "policies/payment-svc.hcl", RulePath: "secret/data/payment-svc/*"},
					Composed:    composed,
				},
				Pass: true,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	if !strings.Contains(out, "OK ") {
		t.Errorf("should show OK line, got:\n%s", out)
	}
	if !strings.Contains(out, "contributions:") {
		t.Errorf("verbose + multi-policy pass should render contributions, got:\n%s", out)
	}
	if !strings.Contains(out, "readonly (secret/*) granted [list read]") {
		t.Errorf("missing readonly contribution line, got:\n%s", out)
	}
	if !strings.Contains(out, "payment-svc (secret/data/payment-svc/*) granted [list read]") {
		t.Errorf("missing payment-svc contribution line, got:\n%s", out)
	}
	// Verbose also prints the explanation trace; must not duplicate the
	// contributions block.
	if got := strings.Count(out, "contributions:"); got != 1 {
		t.Errorf("contributions block rendered %d times, want 1, got:\n%s", got, out)
	}
}

// TestTerminal_Report_NonVerbosePassMultiPolicy_Silent confirms we do not
// spam the default (non-verbose) output with provenance on every passing
// multi-policy test. Users must opt in via -v for that trace.
func TestTerminal_Report_NonVerbosePassMultiPolicy_Silent(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, false)

	composed := &evaluator.Composed{
		Path:    "secret/data/payment-svc/db-creds",
		Granted: map[string]bool{"read": true, "list": true},
		Contributions: []evaluator.RuleContribution{
			{PolicyFile: "policies/readonly.hcl", RulePath: "secret/*", Capabilities: []string{"read", "list"}},
			{PolicyFile: "policies/payment-svc.hcl", RulePath: "secret/data/payment-svc/*", Capabilities: []string{"read", "list"}},
		},
	}

	suite := evaluator.SuiteResult{
		Suite:  "quiet-pass",
		Passed: 1,
		Results: []evaluator.TestResult{
			{
				Test: spec.TestCase{Name: "read payment secrets", Path: "secret/data/payment-svc/db-creds", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{
					TestName: "read payment secrets",
					Path:     "secret/data/payment-svc/db-creds",
					Allowed:  true,
					Composed: composed,
				},
				Pass: true,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	if strings.Contains(out, "contributions:") {
		t.Errorf("non-verbose passing test should stay silent about provenance, got:\n%s", out)
	}
}

// TestTerminal_Report_NoComposed_NoContributions guards against nil
// dereferences when a Result has no Composed field populated (e.g. legacy
// callers or test fixtures built before the composer landed).
func TestTerminal_Report_NoComposed_NoContributions(t *testing.T) {
	var buf bytes.Buffer
	r := NewTerminal(&buf, true)

	suite := evaluator.SuiteResult{
		Suite:  "no-composed",
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test: spec.TestCase{Name: "t", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{
					TestName:    "t",
					Path:        "secret/foo",
					Allowed:     false,
					Explanation: "no policy rule matches path (implicit deny)",
					MatchedRule: nil,
					Composed:    nil,
				},
				Pass: false,
			},
		},
	}

	// Must not panic and must not render contributions.
	r.Report(suite)
	out := buf.String()

	if strings.Contains(out, "contributions:") {
		t.Errorf("nil Composed must not render contributions, got:\n%s", out)
	}
	if !strings.Contains(out, "FAIL ") {
		t.Errorf("should still render FAIL line, got:\n%s", out)
	}
}

func TestTerminal_Report_NOCOLORRespected(t *testing.T) {
	// fatih/color respects NO_COLOR env var natively.
	// Verify that the color.NoColor flag is checkable.
	original := color.NoColor
	defer func() { color.NoColor = original }()

	color.NoColor = true
	var buf bytes.Buffer
	r := NewTerminal(&buf, false)

	suite := evaluator.SuiteResult{
		Suite:  "nocolor-suite",
		Passed: 1,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "test", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{TestName: "test", Path: "secret/foo", Allowed: true, Explanation: "ok"},
				Pass:   true,
			},
		},
	}

	r.Report(suite)
	out := buf.String()

	// With NO_COLOR, output should not contain ANSI escape codes.
	if strings.Contains(out, "\033[") {
		t.Errorf("NO_COLOR should suppress ANSI codes, got:\n%q", out)
	}
}
