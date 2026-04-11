package cmd

import (
	"strings"
	"testing"

	cli "github.com/timkrebs/gocli"
)

func TestVersionCmd_Run(t *testing.T) {
	ui := cli.NewMockUi()
	cmd := &VersionCmd{UI: ui}

	code := cmd.Run(nil)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	out := ui.OutputWriter.String()
	if !strings.Contains(out, "custos v") {
		t.Errorf("output = %q, want to contain %q", out, "custos v")
	}
}

func TestVersionCmd_RunJSON(t *testing.T) {
	ui := cli.NewMockUi()
	cmd := &VersionCmd{UI: ui}

	code := cmd.Run([]string{"--json"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	out := ui.OutputWriter.String()
	if !strings.Contains(out, `"version"`) {
		t.Errorf("JSON output missing 'version' key: %q", out)
	}
	if !strings.Contains(out, `"git_commit"`) {
		t.Errorf("JSON output missing 'git_commit' key: %q", out)
	}
}

func TestVersionCmd_InvalidFlag(t *testing.T) {
	ui := cli.NewMockUi()
	cmd := &VersionCmd{UI: ui}

	code := cmd.Run([]string{"--invalid"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestVersionCmd_Name(t *testing.T) {
	cmd := &VersionCmd{}
	if cmd.Name() != "version" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "version")
	}
}

func TestVersionCmd_Synopsis(t *testing.T) {
	cmd := &VersionCmd{}
	if cmd.Synopsis() == "" {
		t.Error("Synopsis() should not be empty")
	}
}

func TestVersionCmd_Help(t *testing.T) {
	cmd := &VersionCmd{}
	help := cmd.Help()
	if !strings.Contains(help, "custos version") {
		t.Errorf("Help() = %q, want to contain %q", help, "custos version")
	}
}
