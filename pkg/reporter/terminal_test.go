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
