package cmd

import (
	"bytes"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
	cli "github.com/timkrebs/gocli"
)

func init() {
	// Disable colors in tests for predictable output assertions.
	color.NoColor = true
}

// helper to write a policy and spec to a temp dir, returning the spec path.
func writeFixture(t *testing.T, policyHCL, specYAML string) (specPath string) {
	t.Helper()
	tmpDir := t.TempDir()

	policyFile := filepath.Join(tmpDir, "policy.hcl")
	if err := os.WriteFile(policyFile, []byte(policyHCL), 0644); err != nil {
		t.Fatal(err)
	}

	// Replace placeholder with actual policy path.
	specYAML = strings.ReplaceAll(specYAML, "POLICY_PATH", policyFile)
	specFile := filepath.Join(tmpDir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte(specYAML), 0644); err != nil {
		t.Fatal(err)
	}
	return specFile
}

func TestCliStartCmd_Run_MissingFlag(t *testing.T) {
	ui := cli.NewMockUi()
	cmd := &CliStartCmd{UI: ui}

	code := cmd.Run(nil)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (missing -f flag)", code)
	}
	errOut := ui.ErrorWriter.String()
	if !strings.Contains(errOut, "-f") && !strings.Contains(errOut, "--file") {
		t.Errorf("error output should mention -f flag, got: %s", errOut)
	}
}

func TestCliStartCmd_Run_InvalidSpecFile(t *testing.T) {
	ui := cli.NewMockUi()
	cmd := &CliStartCmd{UI: ui}

	code := cmd.Run([]string{"-f", "nonexistent.yaml"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (invalid spec file)", code)
	}
	errOut := ui.ErrorWriter.String()
	if !strings.Contains(errOut, "Error loading spec") {
		t.Errorf("should show clear error for missing spec, got: %s", errOut)
	}
}

func TestCliStartCmd_Run_InvalidPolicyFile(t *testing.T) {
	tmpDir := t.TempDir()
	specContent := `
suite: "bad-policy"
policies:
  - path: /nonexistent/policy.hcl
tests:
  - name: "test"
    path: "secret/foo"
    capabilities: [read]
    expect: allow
`
	specFile := filepath.Join(tmpDir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}

	ui := cli.NewMockUi()
	var buf bytes.Buffer
	cmd := &CliStartCmd{UI: ui, Writer: &buf}

	code := cmd.Run([]string{"-f", specFile})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (invalid policy file)", code)
	}
	errOut := ui.ErrorWriter.String()
	if !strings.Contains(errOut, "Error parsing policy") {
		t.Errorf("should show clear error for missing policy, got: %s", errOut)
	}
}

func TestCliStartCmd_Run_AllPass(t *testing.T) {
	specFile := writeFixture(t,
		`path "secret/foo" { capabilities = ["read"] }`,
		`
suite: "test-suite"
policies:
  - path: POLICY_PATH
tests:
  - name: "allow read"
    path: "secret/foo"
    capabilities: [read]
    expect: allow
  - name: "implicit deny"
    path: "secret/bar"
    capabilities: [read]
    expect: deny
`)

	ui := cli.NewMockUi()
	var buf bytes.Buffer
	cmd := &CliStartCmd{UI: ui, Writer: &buf}

	code := cmd.Run([]string{"-f", specFile})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, ui.ErrorWriter.String())
	}

	out := buf.String()
	if !strings.Contains(out, "OK ") {
		t.Errorf("output should contain OK, got:\n%s", out)
	}
	if !strings.Contains(out, "test-suite") {
		t.Errorf("output should contain suite name, got:\n%s", out)
	}
	if !strings.Contains(out, "2 passed") {
		t.Errorf("output should show 2 passed, got:\n%s", out)
	}
	if !strings.Contains(out, "0 failed") {
		t.Errorf("output should show 0 failed, got:\n%s", out)
	}
}

func TestCliStartCmd_Run_WithFailure(t *testing.T) {
	specFile := writeFixture(t,
		`path "secret/foo" { capabilities = ["read"] }`,
		`
suite: "failing-suite"
policies:
  - path: POLICY_PATH
tests:
  - name: "should fail"
    path: "secret/foo"
    capabilities: [create]
    expect: allow
`)

	ui := cli.NewMockUi()
	var buf bytes.Buffer
	cmd := &CliStartCmd{UI: ui, Writer: &buf}

	code := cmd.Run([]string{"-f", specFile})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}

	out := buf.String()
	if !strings.Contains(out, "FAIL ") {
		t.Errorf("output should contain FAIL, got:\n%s", out)
	}
	if !strings.Contains(out, "1 failed") {
		t.Errorf("should show 1 failed, got:\n%s", out)
	}
}

func TestCliStartCmd_Run_VerboseFlag(t *testing.T) {
	specFile := writeFixture(t,
		`path "secret/foo" { capabilities = ["read"] }`,
		`
suite: "verbose-suite"
policies:
  - path: POLICY_PATH
tests:
  - name: "allow read"
    path: "secret/foo"
    capabilities: [read]
    expect: allow
`)

	ui := cli.NewMockUi()
	var buf bytes.Buffer
	cmd := &CliStartCmd{UI: ui, Writer: &buf}

	code := cmd.Run([]string{"-f", specFile, "-v"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, ui.ErrorWriter.String())
	}

	out := buf.String()
	if !strings.Contains(out, "allowed by rule") {
		t.Errorf("verbose output should contain explanation, got:\n%s", out)
	}
}

func TestCliStartCmd_Run_FailOnWarn_NoWarnings(t *testing.T) {
	specFile := writeFixture(t,
		`path "secret/foo" { capabilities = ["read"] }`,
		`
suite: "warn-suite"
policies:
  - path: POLICY_PATH
tests:
  - name: "allow read"
    path: "secret/foo"
    capabilities: [read]
    expect: allow
`)

	ui := cli.NewMockUi()
	var buf bytes.Buffer
	cmd := &CliStartCmd{UI: ui, Writer: &buf}

	// --fail-on-warn with no warnings should still exit 0.
	code := cmd.Run([]string{"-f", specFile, "--fail-on-warn"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (no warnings)", code)
	}
}

func TestCliStartCmd_Run_DenyOverride(t *testing.T) {
	specFile := writeFixture(t,
		`
path "secret/data/billing/*" { capabilities = ["deny"] }
path "secret/data/app/*" { capabilities = ["read", "list"] }
`,
		`
suite: "deny-override"
policies:
  - path: POLICY_PATH
tests:
  - name: "billing denied"
    path: "secret/data/billing/key"
    capabilities: [read]
    expect: deny
  - name: "app allowed"
    path: "secret/data/app/config"
    capabilities: [read]
    expect: allow
`)

	ui := cli.NewMockUi()
	var buf bytes.Buffer
	cmd := &CliStartCmd{UI: ui, Writer: &buf}

	code := cmd.Run([]string{"-f", specFile})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s\nout: %s", code, ui.ErrorWriter.String(), buf.String())
	}

	out := buf.String()
	if strings.Count(out, "OK ") != 2 {
		t.Errorf("expected 2 OK lines, got:\n%s", out)
	}
}

func TestCliStartCmd_Run_MultipleCapabilities(t *testing.T) {
	specFile := writeFixture(t,
		`path "pki/issue/app" { capabilities = ["create", "update"] }`,
		`
suite: "multi-cap"
policies:
  - path: POLICY_PATH
tests:
  - name: "can issue certs"
    path: "pki/issue/app"
    capabilities: [create, update]
    expect: allow
  - name: "cannot delete"
    path: "pki/issue/app"
    capabilities: [delete]
    expect: deny
`)

	ui := cli.NewMockUi()
	var buf bytes.Buffer
	cmd := &CliStartCmd{UI: ui, Writer: &buf}

	code := cmd.Run([]string{"-f", specFile})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nout: %s", code, buf.String())
	}
}

func TestCliStartCmd_Name(t *testing.T) {
	cmd := &CliStartCmd{}
	if cmd.Name() != "test" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "test")
	}
}

func TestCliStartCmd_Synopsis(t *testing.T) {
	cmd := &CliStartCmd{}
	if cmd.Synopsis() == "" {
		t.Error("Synopsis() should not be empty")
	}
}

// junitDoc is a minimal tree used to validate that the CLI end-to-end
// pipeline emits parseable JUnit XML. It purposefully duplicates a subset
// of pkg/reporter internals so the CLI test does not depend on unexported
// types from that package.
type junitDoc struct {
	XMLName  xml.Name `xml:"testsuites"`
	Name     string   `xml:"name,attr"`
	Tests    int      `xml:"tests,attr"`
	Failures int      `xml:"failures,attr"`
	Suites   []struct {
		Name      string `xml:"name,attr"`
		Tests     int    `xml:"tests,attr"`
		Failures  int    `xml:"failures,attr"`
		Testcases []struct {
			Name    string `xml:"name,attr"`
			Failure *struct {
				Message string `xml:"message,attr"`
				Type    string `xml:"type,attr"`
				Content string `xml:",chardata"`
			} `xml:"failure,omitempty"`
		} `xml:"testcase"`
	} `xml:"testsuite"`
}

// TestCliStartCmd_Run_JUnitFormat_AllPass asserts that --format=junit
// produces a parseable XML document through the full load -> parse ->
// evaluate -> report pipeline. This is the acceptance test for the
// feature: if encoding/xml reparses the CLI's stdout into a valid
// junitDoc, dorny/test-reporter and similar CI tools will also accept it.
func TestCliStartCmd_Run_JUnitFormat_AllPass(t *testing.T) {
	specFile := writeFixture(t,
		`path "secret/foo" { capabilities = ["read"] }`,
		`
suite: "junit-pass-suite"
policies:
  - path: POLICY_PATH
tests:
  - name: "allow read"
    path: "secret/foo"
    capabilities: [read]
    expect: allow
`)

	ui := cli.NewMockUi()
	var buf bytes.Buffer
	cmd := &CliStartCmd{UI: ui, Writer: &buf}

	code := cmd.Run([]string{"-f", specFile, "--format=junit"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s\nstdout: %s", code, ui.ErrorWriter.String(), buf.String())
	}

	out := buf.String()
	if !strings.HasPrefix(out, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Errorf("junit output missing XML declaration, got prefix: %q", out[:min(60, len(out))])
	}
	if strings.Contains(out, "OK ") || strings.Contains(out, "FAIL ") {
		t.Errorf("junit output should not contain terminal markers, got:\n%s", out)
	}

	var doc junitDoc
	if err := xml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("junit output not parseable: %v\n%s", err, out)
	}
	if doc.Tests != 1 || doc.Failures != 0 {
		t.Errorf("counts wrong: tests=%d failures=%d", doc.Tests, doc.Failures)
	}
	if len(doc.Suites) != 1 || len(doc.Suites[0].Testcases) != 1 {
		t.Fatalf("unexpected doc shape: %+v", doc)
	}
	if doc.Suites[0].Name != "junit-pass-suite" {
		t.Errorf("testsuite name = %q, want %q", doc.Suites[0].Name, "junit-pass-suite")
	}
	if doc.Suites[0].Testcases[0].Failure != nil {
		t.Errorf("passing test should not carry <failure>")
	}
}

// TestCliStartCmd_Run_JUnitFormat_WithFailure covers the same pipeline
// for a failing case and asserts that the failure element carries the
// path, expected/got, and policy attribution that dorny/test-reporter
// surfaces in the dashboard.
func TestCliStartCmd_Run_JUnitFormat_WithFailure(t *testing.T) {
	specFile := writeFixture(t,
		`path "secret/foo" { capabilities = ["read"] }`,
		`
suite: "junit-fail-suite"
policies:
  - path: POLICY_PATH
tests:
  - name: "wants create"
    path: "secret/foo"
    capabilities: [create]
    expect: allow
`)

	ui := cli.NewMockUi()
	var buf bytes.Buffer
	cmd := &CliStartCmd{UI: ui, Writer: &buf}

	code := cmd.Run([]string{"-f", specFile, "--format=junit"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (failure)\nstderr: %s\nstdout: %s", code, ui.ErrorWriter.String(), buf.String())
	}

	var doc junitDoc
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("junit output not parseable: %v\n%s", err, buf.String())
	}
	if doc.Failures != 1 {
		t.Errorf("failures = %d, want 1", doc.Failures)
	}
	if len(doc.Suites) != 1 || len(doc.Suites[0].Testcases) != 1 {
		t.Fatalf("unexpected doc shape: %+v", doc)
	}
	tc := doc.Suites[0].Testcases[0]
	if tc.Failure == nil {
		t.Fatal("failing testcase missing <failure> element")
	}
	if !strings.Contains(tc.Failure.Message, "expected allow, got deny") {
		t.Errorf("failure message missing expected/got, got: %q", tc.Failure.Message)
	}
	if !strings.Contains(tc.Failure.Message, "secret/foo") {
		t.Errorf("failure message missing path, got: %q", tc.Failure.Message)
	}
	if !strings.Contains(tc.Failure.Content, "Path:     secret/foo") {
		t.Errorf("failure body missing path line, got:\n%s", tc.Failure.Content)
	}
	if !strings.Contains(tc.Failure.Content, "Capabilities: [create]") {
		t.Errorf("failure body missing capabilities line, got:\n%s", tc.Failure.Content)
	}
}

// cliFailingWriter implements io.Writer and returns an error on the very
// first Write call. Used to cover the reporter-write error path in
// CliStartCmd.Run when the caller supplies a broken Writer (for example,
// a closed pipe or a disk-full redirect target).
type cliFailingWriter struct{}

func (cliFailingWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("cli: simulated writer error")
}

// TestCliStartCmd_Run_ReportWriteError injects a failing writer and runs
// the pipeline with --format=junit so the reporter attempts a real write
// and surfaces an error. The CLI must exit 1 and emit the wrapped
// "Error writing report" message to stderr so users can distinguish
// report-emission failures from evaluation failures.
func TestCliStartCmd_Run_ReportWriteError(t *testing.T) {
	specFile := writeFixture(t,
		`path "secret/foo" { capabilities = ["read"] }`,
		`
suite: "write-error-suite"
policies:
  - path: POLICY_PATH
tests:
  - name: "t"
    path: "secret/foo"
    capabilities: [read]
    expect: allow
`)

	ui := cli.NewMockUi()
	cmd := &CliStartCmd{UI: ui, Writer: cliFailingWriter{}}

	code := cmd.Run([]string{"-f", specFile, "--format=junit"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 on reporter write failure", code)
	}
	errOut := ui.ErrorWriter.String()
	if !strings.Contains(errOut, "Error writing report") {
		t.Errorf("stderr should mention 'Error writing report', got: %s", errOut)
	}
}

// TestCliStartCmd_Run_InvalidFormat_ErrorsWithHelpfulMessage checks that
// an unknown --format value fails fast with an error message listing the
// supported values.
func TestCliStartCmd_Run_InvalidFormat_ErrorsWithHelpfulMessage(t *testing.T) {
	specFile := writeFixture(t,
		`path "secret/foo" { capabilities = ["read"] }`,
		`
suite: "bad-format"
policies:
  - path: POLICY_PATH
tests:
  - name: "t"
    path: "secret/foo"
    capabilities: [read]
    expect: allow
`)

	ui := cli.NewMockUi()
	var buf bytes.Buffer
	cmd := &CliStartCmd{UI: ui, Writer: &buf}

	code := cmd.Run([]string{"-f", specFile, "--format=bogus"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	errOut := ui.ErrorWriter.String()
	if !strings.Contains(errOut, "bogus") {
		t.Errorf("error should echo invalid format, got: %s", errOut)
	}
	if !strings.Contains(errOut, "terminal") || !strings.Contains(errOut, "junit") {
		t.Errorf("error should list supported formats, got: %s", errOut)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestCliStartCmd_Help(t *testing.T) {
	cmd := &CliStartCmd{}
	help := cmd.Help()
	if !strings.Contains(help, "custos test") {
		t.Errorf("Help() should contain 'custos test', got: %s", help)
	}
	if !strings.Contains(help, "--file") {
		t.Errorf("Help() should mention --file flag")
	}
	if !strings.Contains(help, "--fail-on-warn") {
		t.Errorf("Help() should mention --fail-on-warn flag")
	}
	if !strings.Contains(help, "--verbose") {
		t.Errorf("Help() should mention --verbose flag")
	}
	if !strings.Contains(help, "--format") {
		t.Errorf("Help() should mention --format flag")
	}
	if !strings.Contains(help, "junit") {
		t.Errorf("Help() should mention junit format option")
	}
}
