package cmd

import (
	"strings"
	"testing"

	cli "github.com/timkrebs/gocli"
)

func TestCliStartCmd_Run(t *testing.T) {
	ui := cli.NewMockUi()
	cmd := &CliStartCmd{UI: ui}

	code := cmd.Run(nil)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
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
