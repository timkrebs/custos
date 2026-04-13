package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cli "github.com/timkrebs/gocli"
)

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
}

func TestCliStartCmd_Run_ValidSpec(t *testing.T) {
	// Write a temporary spec and policy for the test.
	tmpDir := t.TempDir()

	policyContent := `
path "secret/foo" {
  capabilities = ["read"]
}
`
	policyFile := filepath.Join(tmpDir, "policy.hcl")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	specContent := `
suite: "test-suite"
policies:
  - path: ` + policyFile + `
tests:
  - name: "allow read"
    path: "secret/foo"
    capabilities: [read]
    expect: allow
  - name: "implicit deny"
    path: "secret/bar"
    capabilities: [read]
    expect: deny
`
	specFile := filepath.Join(tmpDir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}

	ui := cli.NewMockUi()
	cmd := &CliStartCmd{UI: ui}

	code := cmd.Run([]string{"-f", specFile})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, ui.ErrorWriter.String())
	}

	out := ui.OutputWriter.String()
	if !strings.Contains(out, "PASS") {
		t.Errorf("output should contain PASS, got: %s", out)
	}
	if !strings.Contains(out, "SUITE") {
		t.Errorf("output should contain SUITE header, got: %s", out)
	}
}

func TestCliStartCmd_Run_FailingTest(t *testing.T) {
	tmpDir := t.TempDir()

	policyContent := `
path "secret/foo" {
  capabilities = ["read"]
}
`
	policyFile := filepath.Join(tmpDir, "policy.hcl")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatal(err)
	}

	// This spec expects "allow" but the policy doesn't grant "create"
	specContent := `
suite: "failing-suite"
policies:
  - path: ` + policyFile + `
tests:
  - name: "should fail"
    path: "secret/foo"
    capabilities: [create]
    expect: allow
`
	specFile := filepath.Join(tmpDir, "spec.yaml")
	if err := os.WriteFile(specFile, []byte(specContent), 0644); err != nil {
		t.Fatal(err)
	}

	ui := cli.NewMockUi()
	cmd := &CliStartCmd{UI: ui}

	code := cmd.Run([]string{"-f", specFile})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (test should fail)", code)
	}

	errOut := ui.ErrorWriter.String()
	if !strings.Contains(errOut, "FAIL") {
		t.Errorf("error output should contain FAIL, got: %s", errOut)
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
		t.Errorf("Help() = %q, want to contain %q", help, "custos test")
	}
	if !strings.Contains(help, "--file") {
		t.Errorf("Help() should mention --file flag")
	}
}
