package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/timkrebs/custos/pkg/evaluator"
	"github.com/timkrebs/custos/pkg/spec"
)

// ---------------------------------------------------------------------------
// Schema shape and validity
// ---------------------------------------------------------------------------

// TestJSON_Report_EmitsValidSchema round-trips a representative suite
// through encoding/json and asserts every top-level field is present
// with the expected type. If consumers use jq or a typed deserializer
// against this document, this test is their contract.
func TestJSON_Report_EmitsValidSchema(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSON(&buf, true)

	suite := evaluator.SuiteResult{
		Suite:    "payment-service-policies",
		Passed:   1,
		Failed:   1,
		Duration: 2500 * time.Microsecond,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "allow read", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{TestName: "allow read", Path: "secret/foo", Allowed: true, Explanation: "ok", MatchedRule: &evaluator.MatchedRule{PolicyFile: "policies/app.hcl", RulePath: "secret/foo", Capabilities: []string{"read"}}},
				Pass:   true, Duration: 1 * time.Millisecond,
			},
			{
				Test:   spec.TestCase{Name: "wants create", Path: "secret/foo", Capabilities: []string{"create"}, Expect: "allow"},
				Result: evaluator.Result{TestName: "wants create", Path: "secret/foo", Allowed: false, Explanation: "missing create"},
				Pass:   false, Duration: 1500 * time.Microsecond,
			},
		},
	}

	if err := r.Report(suite); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	var doc jsonDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("emitted JSON is not parseable: %v\n%s", err, buf.String())
	}

	if doc.SchemaVersion != JSONSchemaVersion {
		t.Errorf("schema_version = %q, want %q", doc.SchemaVersion, JSONSchemaVersion)
	}
	if doc.Suite != "payment-service-policies" {
		t.Errorf("suite = %q", doc.Suite)
	}
	if doc.Summary.Total != 2 || doc.Summary.Passed != 1 || doc.Summary.Failed != 1 {
		t.Errorf("summary counts wrong: %+v", doc.Summary)
	}
	if doc.Duration <= 0 {
		t.Errorf("duration_seconds should be > 0, got %v", doc.Duration)
	}
	if len(doc.Results) != 2 {
		t.Fatalf("results len = %d, want 2", len(doc.Results))
	}

	pass := doc.Results[0]
	if pass.Name != "allow read" || !pass.Pass || pass.Expected != "allow" || pass.Actual != "allow" {
		t.Errorf("passing result wrong: %+v", pass)
	}
	if pass.MatchedRule == nil || pass.MatchedRule.RulePath != "secret/foo" {
		t.Errorf("passing result missing matched_rule: %+v", pass.MatchedRule)
	}

	fail := doc.Results[1]
	if fail.Pass || fail.Expected != "allow" || fail.Actual != "deny" {
		t.Errorf("failing result wrong: %+v", fail)
	}
}

// TestJSON_Report_RawKeysPresent verifies the on-the-wire field names are
// the snake_case forms consumers depend on. Renaming any of these fields
// is a breaking change; this test guards against accidental renames.
func TestJSON_Report_RawKeysPresent(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSON(&buf, true)

	suite := evaluator.SuiteResult{
		Suite:  "raw-keys",
		Passed: 1,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "t", Path: "p", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{Allowed: true, MatchedRule: &evaluator.MatchedRule{PolicyFile: "a.hcl", RulePath: "p", Capabilities: []string{"read"}}},
				Pass:   true,
			},
		},
	}
	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	mustContain := []string{
		`"schema_version"`,
		`"suite"`,
		`"duration_seconds"`,
		`"summary"`,
		`"total"`,
		`"passed"`,
		`"failed"`,
		`"warnings"`,
		`"results"`,
		`"name"`,
		`"path"`,
		`"capabilities"`,
		`"expected"`,
		`"actual"`,
		`"pass"`,
		`"explanation"`,
		`"matched_rule"`,
		`"composed"`,
		`"policy_file"`,
		`"rule_path"`,
	}
	for _, key := range mustContain {
		if !strings.Contains(out, key) {
			t.Errorf("missing JSON key %s in output:\n%s", key, out)
		}
	}
}

// ---------------------------------------------------------------------------
// Pretty vs compact
// ---------------------------------------------------------------------------

// TestJSON_Report_PrettyOutputIsIndented confirms the default (Pretty=true)
// path produces a multi-line, indented document. A human reading the
// output should be able to scan the tree without tooling.
func TestJSON_Report_PrettyOutputIsIndented(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSON(&buf, true)

	suite := evaluator.SuiteResult{
		Suite:  "pretty",
		Passed: 1,
		Results: []evaluator.TestResult{
			{Test: spec.TestCase{Name: "t", Path: "p", Expect: "allow"}, Result: evaluator.Result{Allowed: true}, Pass: true},
		},
	}
	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, "\n") {
		t.Errorf("pretty output should span multiple lines, got:\n%s", out)
	}
	if !strings.Contains(out, "  \"schema_version\"") {
		t.Errorf("pretty output should indent nested fields with two spaces, got:\n%s", out)
	}
}

// TestJSON_Report_CompactOutputIsOneLine asserts that Pretty=false produces
// a single-line document (plus the trailing newline), which is what line-
// oriented tools and log aggregators consume.
func TestJSON_Report_CompactOutputIsOneLine(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSON(&buf, false)

	suite := evaluator.SuiteResult{
		Suite:  "compact",
		Passed: 1,
		Results: []evaluator.TestResult{
			{Test: spec.TestCase{Name: "t", Path: "p", Expect: "allow"}, Result: evaluator.Result{Allowed: true}, Pass: true},
		},
	}
	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	// Exactly one line break, at the very end.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("compact output should be one line, got %d lines:\n%s", len(lines), out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("compact output should end with a newline for line-oriented tools")
	}
	// Compact should still round-trip as valid JSON.
	var doc jsonDocument
	if err := json.Unmarshal([]byte(lines[0]), &doc); err != nil {
		t.Fatalf("compact JSON not parseable: %v\n%s", err, out)
	}
}

// ---------------------------------------------------------------------------
// Provenance rendering
// ---------------------------------------------------------------------------

// TestJSON_Report_ComposedProvenanceInResults flattens the composer's
// multi-policy provenance into the composed sub-object so jq consumers
// can answer questions like "which policies granted read?" without
// re-evaluating the policies.
func TestJSON_Report_ComposedProvenanceInResults(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSON(&buf, true)

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
		Suite:  "provenance",
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "billing denied", Path: "secret/data/billing-svc/api-key", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{Allowed: false, Composed: composed, Explanation: "denied"},
				Pass:   false,
			},
		},
	}

	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}

	var doc jsonDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	c := doc.Results[0].Composed
	if c == nil {
		t.Fatal("composed should not be null when provenance is available")
	}
	if !c.Denied {
		t.Error("composed.denied should be true")
	}

	// Granted must be sorted for deterministic output.
	if len(c.Granted) != 2 || c.Granted[0] != "list" || c.Granted[1] != "read" {
		t.Errorf("granted = %v, want [list read]", c.Granted)
	}
	if len(c.Contributions) != 2 {
		t.Errorf("expected 2 contributions, got %d", len(c.Contributions))
	}
	if len(c.DeniedBy) != 1 || !c.DeniedBy[0].IsDeny {
		t.Errorf("denied_by should contain one is_deny entry, got %+v", c.DeniedBy)
	}

	// Contribution order must match composer order, so consumers can
	// render chronological provenance walks.
	if c.Contributions[0].PolicyFile != "policies/readonly.hcl" {
		t.Errorf("contribution order lost: %+v", c.Contributions)
	}
	if !c.Contributions[1].IsDeny {
		t.Errorf("second contribution should be a deny: %+v", c.Contributions[1])
	}
}

// TestJSON_Report_NilArraysRenderAsEmpty guards the jq-friendliness
// invariant: slice fields are always emitted as arrays, never as null.
// A consumer writing `.results[] | .capabilities[]` should not have to
// check for null before iterating.
func TestJSON_Report_NilArraysRenderAsEmpty(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSON(&buf, false)

	// Result with nil capabilities and no composed provenance.
	suite := evaluator.SuiteResult{
		Suite: "nil-arrays",
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "t", Path: "p", Capabilities: nil, Expect: "allow"},
				Result: evaluator.Result{Allowed: false},
				Pass:   false,
			},
		},
		Warnings: nil,
	}
	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if strings.Contains(out, `"warnings":null`) || strings.Contains(out, `"warnings": null`) {
		t.Errorf("warnings must render as [] not null, got:\n%s", out)
	}
	if strings.Contains(out, `"capabilities":null`) || strings.Contains(out, `"capabilities": null`) {
		t.Errorf("capabilities must render as [] not null, got:\n%s", out)
	}
	// Must still round-trip.
	var doc jsonDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("JSON not parseable: %v\n%s", err, out)
	}
	if doc.Warnings == nil {
		t.Error("Warnings field should deserialize as non-nil empty slice")
	}
	if doc.Results[0].Capabilities == nil {
		t.Error("Capabilities field should deserialize as non-nil empty slice")
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

// TestJSON_Report_EmptySuite ensures a zero-result suite still produces a
// valid document with a schema_version, an empty results array, and
// zero counts. Consumers that run custos on an empty spec should get a
// clean green output, not a parse error.
func TestJSON_Report_EmptySuite(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSON(&buf, true)

	if err := r.Report(evaluator.SuiteResult{Suite: "empty"}); err != nil {
		t.Fatal(err)
	}

	var doc jsonDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("empty suite JSON not parseable: %v\n%s", err, buf.String())
	}
	if doc.Summary.Total != 0 {
		t.Errorf("empty suite total = %d, want 0", doc.Summary.Total)
	}
	if doc.Results == nil {
		t.Error("results should be empty slice, not nil")
	}
}

// TestJSON_Report_SuiteNameFallback covers the unnamed-spec path: when no
// suite name is set, the reporter falls back to "custos" (matching the
// JUnit reporter's convention).
func TestJSON_Report_SuiteNameFallback(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSON(&buf, true)
	if err := r.Report(evaluator.SuiteResult{}); err != nil {
		t.Fatal(err)
	}
	var doc jsonDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Suite != "custos" {
		t.Errorf("fallback suite = %q, want custos", doc.Suite)
	}
}

// TestJSON_Report_StringEscaping ensures characters with special meaning
// in JSON ("\", ", newlines, tabs, control chars) are properly escaped.
// encoding/json handles this correctly by default; the test locks in
// that behavior so a future manual-concat refactor cannot regress it.
func TestJSON_Report_StringEscaping(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSON(&buf, false)

	suite := evaluator.SuiteResult{
		Suite: `weird "name" with \ backslash`,
		Results: []evaluator.TestResult{
			{
				Test:   spec.TestCase{Name: "line1\nline2\tindent", Path: `p/"x"`, Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{Allowed: false, Explanation: "</script><control\u0001>"},
				Pass:   false,
			},
		},
	}
	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}

	// Raw newline must not appear inside a quoted string: Go's
	// encoding/json renders it as \n.
	var doc jsonDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("escaped JSON not parseable: %v\n%s", err, buf.String())
	}
	if doc.Suite != `weird "name" with \ backslash` {
		t.Errorf("suite round-trip lost escaping: %q", doc.Suite)
	}
	if doc.Results[0].Name != "line1\nline2\tindent" {
		t.Errorf("name round-trip lost escaping: %q", doc.Results[0].Name)
	}
	if doc.Results[0].Explanation != "</script><control\u0001>" {
		t.Errorf("explanation round-trip lost escaping: %q", doc.Results[0].Explanation)
	}
}

// TestJSON_Report_WarningsPopulated exercises the top-level warnings
// array when SuiteResult.Warnings is non-empty. The summary count must
// match the slice length so jq filters such as
// '.summary.warnings > 0' remain useful.
func TestJSON_Report_WarningsPopulated(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSON(&buf, true)

	suite := evaluator.SuiteResult{
		Suite:    "warns",
		Warnings: []string{"overly permissive rule", "missing audit log"},
	}
	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}
	var doc jsonDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Warnings) != 2 {
		t.Errorf("warnings len = %d, want 2", len(doc.Warnings))
	}
	if doc.Summary.Warnings != 2 {
		t.Errorf("summary.warnings = %d, want 2", doc.Summary.Warnings)
	}
}

// ---------------------------------------------------------------------------
// Writer failure paths
// ---------------------------------------------------------------------------

// TestJSON_Report_WriteBodyError triggers the main Write error path by
// failing on the first byte. The error must be wrapped with "writing
// JSON" so CLI users see a clear cause.
func TestJSON_Report_WriteBodyError(t *testing.T) {
	fw := &failWriter{failAfter: 0}
	r := NewJSON(fw, true)

	err := r.Report(evaluator.SuiteResult{
		Suite:  "x",
		Passed: 1,
		Results: []evaluator.TestResult{
			{Test: spec.TestCase{Name: "t", Path: "p", Expect: "allow"}, Result: evaluator.Result{Allowed: true}, Pass: true},
		},
	})
	if err == nil {
		t.Fatal("expected error when body write fails")
	}
	if !strings.Contains(err.Error(), "writing JSON") {
		t.Errorf("error should wrap with 'writing JSON', got: %v", err)
	}
}

// TestJSON_Report_WriteNewlineError covers the second error return: the
// body write succeeds, then the trailing newline write fails. Matches
// the JUnit reporter's error-handling discipline.
func TestJSON_Report_WriteNewlineError(t *testing.T) {
	fw := &failWriter{failAfter: 1}
	r := NewJSON(fw, false)

	err := r.Report(evaluator.SuiteResult{
		Suite:  "x",
		Passed: 1,
		Results: []evaluator.TestResult{
			{Test: spec.TestCase{Name: "t", Path: "p", Expect: "allow"}, Result: evaluator.Result{Allowed: true}, Pass: true},
		},
	})
	if err == nil {
		t.Fatal("expected error when trailing newline write fails")
	}
	if !strings.Contains(err.Error(), "trailing newline") {
		t.Errorf("error should wrap with 'trailing newline', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Factory and constructor
// ---------------------------------------------------------------------------

// TestNew_FactoryReturnsJSONReporter locks in that the factory maps the
// "json" format string to a *JSON with Pretty=true by default.
func TestNew_FactoryReturnsJSONReporter(t *testing.T) {
	var buf bytes.Buffer
	r, err := New(FormatJSON, &buf, false)
	if err != nil {
		t.Fatalf("New(json) err = %v", err)
	}
	j, ok := r.(*JSON)
	if !ok {
		t.Fatalf("New(json) returned %T, want *JSON", r)
	}
	if !j.Pretty {
		t.Error("factory should default JSON to pretty=true")
	}
}

func TestNewJSON_NilWriter_DefaultsToStdout(t *testing.T) {
	r := NewJSON(nil, true)
	if r.Writer != os.Stdout {
		t.Error("nil writer should default to os.Stdout")
	}
}

// TestJSON_Report_NilCapabilitiesOnInnerObjects covers the defensive
// nil-to-empty-slice branches inside buildJSONTestResult and
// buildJSONComposed. Exercising them with a MatchedRule and a
// Contribution that both have nil Capabilities flips those branches to
// covered and keeps the "arrays are never null" invariant honest even
// when callers construct Results without explicit slices.
func TestJSON_Report_NilCapabilitiesOnInnerObjects(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSON(&buf, false)

	composed := &evaluator.Composed{
		Granted: map[string]bool{},
		Contributions: []evaluator.RuleContribution{
			{PolicyFile: "a.hcl", RulePath: "secret/foo", Capabilities: nil},
		},
		DeniedBy: []evaluator.RuleContribution{
			{PolicyFile: "b.hcl", RulePath: "secret/bar", Capabilities: nil, IsDeny: true},
		},
		Denied: true,
	}

	suite := evaluator.SuiteResult{
		Suite:  "nil-caps",
		Failed: 1,
		Results: []evaluator.TestResult{
			{
				Test: spec.TestCase{Name: "t", Path: "secret/foo", Capabilities: []string{"read"}, Expect: "allow"},
				Result: evaluator.Result{
					Allowed: false,
					MatchedRule: &evaluator.MatchedRule{
						PolicyFile:   "a.hcl",
						RulePath:     "secret/foo",
						Capabilities: nil,
					},
					Composed: composed,
				},
				Pass: false,
			},
		},
	}

	if err := r.Report(suite); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// No "capabilities":null anywhere.
	if strings.Contains(out, `"capabilities":null`) {
		t.Errorf("capabilities must render as [] even when nil, got:\n%s", out)
	}

	var doc jsonDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("not parseable: %v\n%s", err, out)
	}
	if doc.Results[0].MatchedRule.Capabilities == nil {
		t.Error("matched_rule.capabilities should be empty slice, not nil")
	}
	if doc.Results[0].Composed.Contributions[0].Capabilities == nil {
		t.Error("contribution.capabilities should be empty slice, not nil")
	}
	if doc.Results[0].Composed.DeniedBy[0].Capabilities == nil {
		t.Error("denied_by[].capabilities should be empty slice, not nil")
	}
}
