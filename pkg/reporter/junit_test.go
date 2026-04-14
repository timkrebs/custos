package reporter

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/timkrebs/custos/pkg/evaluator"
	"github.com/timkrebs/custos/pkg/spec"
)

// fixedTime returns a deterministic timestamp source for tests so the
// emitted <testsuite timestamp="..."> attribute is stable across runs.
func fixedTime() time.Time {
	return time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
}

func newFixedJUnit(w io.Writer) *JUnit {
	return &JUnit{Writer: w, Now: fixedTime}
}

// ---------------------------------------------------------------------------
// Document shape and schema validity
// ---------------------------------------------------------------------------

// TestJUnit_Report_EmitsValidXML confirms the emitted document round-trips
// through encoding/xml cleanly. If the bytes reparse into an equivalent
// tree, downstream JUnit consumers (xmllint, dorny/test-reporter, Jenkins)
// will also accept them.
func TestJUnit_Report_EmitsValidXML(t *testing.T) {
	var buf bytes.Buffer
	r := newFixedJUnit(&buf)

	suite := evaluator.SuiteResult{
		Suite:    "payment-service-policies",
		Passed:   1,
		Failed:   0,
		Duration: 5 * time.Millisecond,
		Results: []evaluator.TestResult{
			{
				Test:     spec.TestCase{Name: "allow read", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "allow"},
				Result:   evaluator.Result{TestName: "allow read", Path: "secret/foo", Allowed: true, Explanation: "ok"},
				Pass:     true,
				Duration: 5 * time.Millisecond,
			},
		},
	}

	if err := r.Report(suite); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	out := buf.String()

	// Must begin with the XML declaration.
	if !strings.HasPrefix(out, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Errorf("output missing XML declaration, got prefix: %q", out[:min(60, len(out))])
	}

	// Must round-trip through encoding/xml.
	var parsed junitTestsuites
	if err := xml.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("emitted XML is not parseable: %v\n%s", err, out)
	}
	if parsed.Tests != 1 || parsed.Failures != 0 {
		t.Errorf("roundtrip counts wrong: tests=%d failures=%d", parsed.Tests, parsed.Failures)
	}
	if len(parsed.Suites) != 1 || len(parsed.Suites[0].Testcases) != 1 {
		t.Fatalf("roundtrip shape wrong: %+v", parsed)
	}
	if parsed.Suites[0].Testcases[0].Name != "allow read" {
		t.Errorf("testcase name = %q, want %q", parsed.Suites[0].Testcases[0].Name, "allow read")
	}
	if parsed.Suites[0].Testcases[0].Failure != nil {
		t.Errorf("passing test should have no failure element")
	}
}

// TestJUnit_Report_SchemaAttributesPresent asserts every attribute that
// mainstream CI parsers require is emitted on the root elements.
func TestJUnit_Report_SchemaAttributesPresent(t *testing.T) {
	var buf bytes.Buffer
	r := newFixedJUnit(&buf)

	suite := evaluator.SuiteResult{
		Suite:    "attr-suite",
		Passed:   2,
		Failed:   1,
		Duration: 3*time.Millisecond + 500*time.Microsecond,
		Results: []evaluator.TestResult{
			{Test: spec.TestCase{Name: "a", Path: "p", Expect: "allow"}, Result: evaluator.Result{Allowed: true}, Pass: true, Duration: time.Millisecond},
			{Test: spec.TestCase{Name: "b", Path: "p", Expect: "allow"}, Result: evaluator.Result{Allowed: true}, Pass: true, Duration: time.Millisecond},
			{Test: spec.TestCase{Name: "c", Path: "p", Expect: "allow"}, Result: evaluator.Result{Allowed: false, Explanation: "x"}, Pass: false, Duration: 1500 * time.Microsecond},
		},
	}

	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	mustContain := []string{
		`<testsuites name="custos"`,
		`tests="3"`,
		`failures="1"`,
		`errors="0"`,
		`<testsuite name="attr-suite"`,
		`timestamp="2026-04-14T12:00:00Z"`,
		`classname="attr-suite"`,
		`<failure message="`,
		`type="AssertionError"`,
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\nfull output:\n%s", s, out)
		}
	}
}

// ---------------------------------------------------------------------------
// Failure detail content
// ---------------------------------------------------------------------------

// TestJUnit_Report_FailureIncludesContext verifies the failure chardata
// body carries expected-vs-got, path, capabilities, and the matched rule
// when the evaluator produced one. This is the information CI operators
// need in their drill-down view.
func TestJUnit_Report_FailureIncludesContext(t *testing.T) {
	var buf bytes.Buffer
	r := newFixedJUnit(&buf)

	suite := evaluator.SuiteResult{
		Suite:  "failure-detail",
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test: spec.TestCase{
					Name:         "should allow create",
					Path:         "secret/data/app/key",
					Capabilities: []string{"create"},
					Expect:       "allow",
				},
				Result: evaluator.Result{
					TestName:    "should allow create",
					Path:        "secret/data/app/key",
					Allowed:     false,
					Explanation: "missing capabilities [create] (granted: [read list])",
					MatchedRule: &evaluator.MatchedRule{
						PolicyFile: "policies/readonly.hcl",
						RulePath:   "secret/*",
					},
				},
				Pass: false,
			},
		},
	}

	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}

	// Round-trip through encoding/xml so we assert against the decoded
	// body, not the raw serialized form. encoding/xml escapes quotes in
	// chardata as &#34; which would break a naive substring search even
	// though the downstream consumer sees the unescaped string.
	var parsed junitTestsuites
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("emitted XML not parseable: %v\n%s", err, buf.String())
	}
	if len(parsed.Suites) != 1 || len(parsed.Suites[0].Testcases) != 1 {
		t.Fatalf("unexpected XML shape: %+v", parsed)
	}
	tc := parsed.Suites[0].Testcases[0]
	if tc.Failure == nil {
		t.Fatal("failing testcase has no <failure> child")
	}
	if tc.Failure.Message != "expected allow, got deny at path secret/data/app/key" {
		t.Errorf("failure message = %q", tc.Failure.Message)
	}
	if tc.Failure.Type != "AssertionError" {
		t.Errorf("failure type = %q, want AssertionError", tc.Failure.Type)
	}

	body := tc.Failure.Content
	for _, want := range []string{
		"Expected: allow",
		"Got:      deny",
		"Path:     secret/data/app/key",
		"Capabilities: [create]",
		`Matched rule: "secret/*" in policy "readonly"`,
		"Explanation: missing capabilities",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("failure body missing %q\nfull decoded body:\n%s", want, body)
		}
	}
}

// TestJUnit_Report_MultiPolicyProvenanceInFailure surfaces the composer
// provenance block inside the failure chardata so CI users see which
// policy denied and which policies would have granted access. This is the
// XML analogue of the terminal reporter's contributions: block.
func TestJUnit_Report_MultiPolicyProvenanceInFailure(t *testing.T) {
	var buf bytes.Buffer
	r := newFixedJUnit(&buf)

	composed := &evaluator.Composed{
		Path:    "secret/data/billing-svc/api-key",
		Granted: map[string]bool{"read": true, "list": true},
		Denied:  true,
		Contributions: []evaluator.RuleContribution{
			{PolicyFile: "policies/readonly.hcl", RulePath: "secret/*", Capabilities: []string{"read", "list"}},
			{PolicyFile: "policies/payment-svc.hcl", RulePath: "secret/data/billing-svc/*", Capabilities: []string{"deny"}, IsDeny: true},
		},
	}

	suite := evaluator.SuiteResult{
		Suite:  "provenance",
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test: spec.TestCase{Name: "should allow billing read", Path: "secret/data/billing-svc/api-key", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{
					Allowed:     false,
					Explanation: "explicitly denied",
					MatchedRule: &evaluator.MatchedRule{PolicyFile: "policies/payment-svc.hcl", RulePath: "secret/data/billing-svc/*"},
					Composed:    composed,
				},
				Pass: false,
			},
		},
	}

	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{
		"Contributions:",
		"- readonly (secret/*) granted [list read]",
		"- payment-svc (secret/data/billing-svc/*) DENIED",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("provenance missing %q\n%s", want, out)
		}
	}
}

// TestJUnit_Report_SingleContributionSuppressesProvenance avoids cluttering
// the failure body when only one policy contributed. The primary matched
// rule line already carries that information.
func TestJUnit_Report_SingleContributionSuppressesProvenance(t *testing.T) {
	var buf bytes.Buffer
	r := newFixedJUnit(&buf)

	composed := &evaluator.Composed{
		Granted: map[string]bool{"read": true},
		Contributions: []evaluator.RuleContribution{
			{PolicyFile: "only.hcl", RulePath: "secret/foo", Capabilities: []string{"read"}},
		},
	}

	suite := evaluator.SuiteResult{
		Suite:  "single",
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "wants create", Path: "secret/foo", Capabilities: []string{"create"}, Expect: "allow"},
				Result: evaluator.Result{Allowed: false, MatchedRule: &evaluator.MatchedRule{PolicyFile: "only.hcl", RulePath: "secret/foo"}, Composed: composed},
				Pass:   false,
			},
		},
	}

	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "Contributions:") {
		t.Errorf("single-contribution failure should not render Contributions block\n%s", buf.String())
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

// TestJUnit_Report_EmptySuite must still emit a valid document with zero
// testcases so CI reporters display an empty (green) result instead of
// failing to parse.
func TestJUnit_Report_EmptySuite(t *testing.T) {
	var buf bytes.Buffer
	r := newFixedJUnit(&buf)

	if err := r.Report(evaluator.SuiteResult{Suite: "empty"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	var parsed junitTestsuites
	if err := xml.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("empty suite XML not parseable: %v\n%s", err, out)
	}
	if parsed.Tests != 0 || parsed.Failures != 0 {
		t.Errorf("empty suite counts wrong: tests=%d failures=%d", parsed.Tests, parsed.Failures)
	}
	if len(parsed.Suites) != 1 {
		t.Errorf("expected one testsuite child even when empty, got %d", len(parsed.Suites))
	}
	if len(parsed.Suites[0].Testcases) != 0 {
		t.Errorf("expected zero testcases, got %d", len(parsed.Suites[0].Testcases))
	}
}

// TestJUnit_Report_XMLEscaping ensures characters that are illegal as raw
// XML chardata (< > & " ') are properly escaped by encoding/xml. A naive
// string-concat implementation would emit invalid markup on test names
// containing angle brackets.
func TestJUnit_Report_XMLEscaping(t *testing.T) {
	var buf bytes.Buffer
	r := newFixedJUnit(&buf)

	suite := evaluator.SuiteResult{
		Suite:  "escape-suite",
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test: spec.TestCase{
					Name:         `weird <name> & "quoted"`,
					Path:         `secret/<x>&y`,
					Capabilities: []string{"read"},
					Expect:       "allow",
				},
				Result: evaluator.Result{
					Allowed:     false,
					Explanation: `missing <cap> & such`,
				},
				Pass: false,
			},
		},
	}

	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// Raw angle brackets inside attribute or text values must not leak.
	if strings.Contains(out, `<name>`) {
		t.Errorf("raw <name> leaked into XML, must be escaped:\n%s", out)
	}
	if strings.Contains(out, `&y`) && !strings.Contains(out, `&amp;y`) {
		t.Errorf("& was not escaped:\n%s", out)
	}

	// Round-trip to prove the output is still valid.
	var parsed junitTestsuites
	if err := xml.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("escaped XML not parseable: %v\n%s", err, out)
	}
	if parsed.Suites[0].Testcases[0].Name != `weird <name> & "quoted"` {
		t.Errorf("round-tripped name = %q", parsed.Suites[0].Testcases[0].Name)
	}
}

// TestJUnit_Report_TimingRendersWithMicrosecondPrecision guards the float
// formatting: mainstream JUnit parsers reject scientific notation and
// unit suffixes, so the time attribute must always be plain decimal.
func TestJUnit_Report_TimingRendersWithMicrosecondPrecision(t *testing.T) {
	var buf bytes.Buffer
	r := newFixedJUnit(&buf)

	suite := evaluator.SuiteResult{
		Suite:    "timing",
		Passed:   1,
		Duration: 1234567 * time.Nanosecond, // 0.001234s
		Results: []evaluator.TestResult{
			{
				Test:     spec.TestCase{Name: "fast", Path: "p", Expect: "allow"},
				Result:   evaluator.Result{Allowed: true},
				Pass:     true,
				Duration: 1234567 * time.Nanosecond,
			},
		},
	}

	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, `time="0.001235"`) && !strings.Contains(out, `time="0.001234"`) {
		t.Errorf("expected time attribute with microsecond precision, got:\n%s", out)
	}
	// No scientific notation.
	if strings.Contains(out, "e-") || strings.Contains(out, "E-") {
		t.Errorf("time must not use scientific notation:\n%s", out)
	}
}

// TestJUnit_Report_ZeroDurationFormatted guards the sentinel zero value:
// when a test constructed without timing (zero Duration) is reported, the
// time attribute must still be a valid float, not "0s" or empty.
func TestJUnit_Report_ZeroDurationFormatted(t *testing.T) {
	var buf bytes.Buffer
	r := newFixedJUnit(&buf)

	suite := evaluator.SuiteResult{
		Suite:  "zero-time",
		Passed: 1,
		Results: []evaluator.TestResult{
			{Test: spec.TestCase{Name: "t", Path: "p", Expect: "allow"}, Result: evaluator.Result{Allowed: true}, Pass: true},
		},
	}
	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `time="0.000000"`) {
		t.Errorf("zero duration should render as 0.000000, got:\n%s", buf.String())
	}
}

// TestJUnit_Report_SuiteNameFallback ensures the output remains valid when
// a spec is loaded without a suite name. The fallback value is "custos".
func TestJUnit_Report_SuiteNameFallback(t *testing.T) {
	var buf bytes.Buffer
	r := newFixedJUnit(&buf)

	suite := evaluator.SuiteResult{
		Results: []evaluator.TestResult{
			{Test: spec.TestCase{Name: "t", Path: "p", Expect: "allow"}, Result: evaluator.Result{Allowed: true}, Pass: true},
		},
	}
	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `<testsuite name="custos"`) {
		t.Errorf("missing testsuite fallback name, got:\n%s", buf.String())
	}
}

// ---------------------------------------------------------------------------
// Factory plumbing
// ---------------------------------------------------------------------------

// TestNew_FactorySelectsReporter is a belt-and-braces check on the reporter
// factory: terminal and junit values each return the right concrete type;
// unknown values fail with a message that lists the supported formats.
func TestNew_FactorySelectsReporter(t *testing.T) {
	var buf bytes.Buffer

	term, err := New(FormatTerminal, &buf, false)
	if err != nil {
		t.Fatalf("New(terminal) err = %v", err)
	}
	if _, ok := term.(*Terminal); !ok {
		t.Errorf("New(terminal) returned %T, want *Terminal", term)
	}

	junit, err := New(FormatJUnit, &buf, false)
	if err != nil {
		t.Fatalf("New(junit) err = %v", err)
	}
	if _, ok := junit.(*JUnit); !ok {
		t.Errorf("New(junit) returned %T, want *JUnit", junit)
	}

	// Empty format defaults to terminal.
	def, err := New("", &buf, false)
	if err != nil {
		t.Fatalf("New(empty) err = %v", err)
	}
	if _, ok := def.(*Terminal); !ok {
		t.Errorf("empty format should default to Terminal, got %T", def)
	}

	// Unknown format errors with a helpful message.
	_, err = New("bogus", &buf, false)
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "bogus") || !strings.Contains(err.Error(), "terminal") || !strings.Contains(err.Error(), "junit") {
		t.Errorf("error message should mention bogus + supported values, got: %v", err)
	}
}

func TestNewJUnit_NilWriter_DefaultsToStdout(t *testing.T) {
	r := NewJUnit(nil)
	if r.Writer != os.Stdout {
		t.Error("nil writer should default to os.Stdout")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
