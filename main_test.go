package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var testBinaryPath string

// TestMain builds the binary once and runs all tests against it.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "custos-test-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmpDir)

	binaryName := "custos-test"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	testBinaryPath = filepath.Join(tmpDir, binaryName)

	if out, err := exec.Command("go", "build", "-o", testBinaryPath, ".").CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}

	os.Exit(m.Run())
}

func TestBinary_NoArgs(t *testing.T) {
	cmd := exec.Command(testBinaryPath)
	out, err := cmd.CombinedOutput()
	// No args prints help and exits 1
	if err == nil {
		t.Log("expected non-zero exit for no args")
	}
	if len(out) == 0 {
		t.Error("expected help output, got empty")
	}
}

func TestBinary_Version(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("custos version failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "custos v") {
		t.Errorf("output = %q, want to contain %q", string(out), "custos v")
	}
}

func TestBinary_VersionJSON(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "version", "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("custos version --json failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), `"version"`) {
		t.Errorf("JSON output missing 'version' key: %q", string(out))
	}
}

func TestBinary_Help(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "--help")
	out, err := cmd.CombinedOutput()
	// --help exits 0
	if err != nil {
		t.Logf("custos --help exited with error: %v", err)
	}
	if !strings.Contains(string(out), "version") {
		t.Errorf("help output should list 'version' command: %q", string(out))
	}
}

func TestBinary_UnknownCommand(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "nonexistent")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for unknown command")
	}
	if len(out) == 0 {
		t.Error("expected error output, got empty")
	}
}
