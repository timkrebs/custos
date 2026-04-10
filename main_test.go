package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var testBinaryPath string

// TestMain builds the binary once and runs all tests against it.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "vaultspec-test-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmpDir)

	binaryName := "vaultspec-test"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	testBinaryPath = filepath.Join(tmpDir, binaryName)

	if out, err := exec.Command("go", "build", "-o", testBinaryPath, ".").CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}

	os.Exit(m.Run())
}
