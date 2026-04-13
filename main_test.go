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
	if !strings.Contains(string(out), "test") {
		t.Errorf("help output should list 'test' command: %q", string(out))
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

// ---------------------------------------------------------------------------
// End-to-end: custos test
// ---------------------------------------------------------------------------

func TestBinary_Test_MissingFileFlag(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "test")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit when -f is missing")
	}
	if !strings.Contains(string(out), "-f") {
		t.Errorf("should mention -f flag, got: %s", out)
	}
}

func TestBinary_Test_MissingSpecFile(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "test", "-f", "/nonexistent/spec.yaml")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for missing spec file")
	}
	if !strings.Contains(string(out), "Error loading spec") {
		t.Errorf("should show clear error, got: %s", out)
	}
}

func TestBinary_Test_EndToEnd_AllPass(t *testing.T) {
	// Use the project's own testdata fixtures.
	specPath := filepath.Join("testdata", "specs", "payment-svc.spec.yaml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Skip("testdata fixture not found")
	}

	cmd := exec.Command(testBinaryPath, "test", "-f", specPath)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0 (all pass), got error: %v\n%s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "payment-service-policies") {
		t.Errorf("should show suite name, got:\n%s", output)
	}
	if !strings.Contains(output, "OK") {
		t.Errorf("should show OK lines, got:\n%s", output)
	}
	if !strings.Contains(output, "10 passed") {
		t.Errorf("should show 10 passed, got:\n%s", output)
	}
	if !strings.Contains(output, "0 failed") {
		t.Errorf("should show 0 failed, got:\n%s", output)
	}
}

func TestBinary_Test_EndToEnd_Verbose(t *testing.T) {
	specPath := filepath.Join("testdata", "specs", "payment-svc.spec.yaml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Skip("testdata fixture not found")
	}

	cmd := exec.Command(testBinaryPath, "test", "-f", specPath, "-v")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\n%s", err, out)
	}

	output := string(out)
	// Verbose mode should show evaluation explanations.
	if !strings.Contains(output, "allowed by rule") {
		t.Errorf("verbose should show explanations, got:\n%s", output)
	}
	if !strings.Contains(output, "implicit deny") {
		t.Errorf("verbose should show implicit deny, got:\n%s", output)
	}
}

func TestBinary_Test_EndToEnd_FailOnWarn(t *testing.T) {
	specPath := filepath.Join("testdata", "specs", "payment-svc.spec.yaml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Skip("testdata fixture not found")
	}

	// --fail-on-warn with no warnings should still exit 0.
	cmd := exec.Command(testBinaryPath, "test", "-f", specPath, "--fail-on-warn")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0 (no warnings), got error: %v\n%s", err, out)
	}
}
