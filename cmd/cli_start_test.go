package cmd

import (
	"bytes"
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
}
