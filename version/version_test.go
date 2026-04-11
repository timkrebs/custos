package version

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
)

func TestGetInfo(t *testing.T) {
	info := GetInfo()

	if info.Version != Version {
		t.Errorf("Version = %q, want %q", info.Version, Version)
	}
	if info.GitCommit != GitCommit {
		t.Errorf("GitCommit = %q, want %q", info.GitCommit, GitCommit)
	}
	if info.GitTreeState != GitTreeState {
		t.Errorf("GitTreeState = %q, want %q", info.GitTreeState, GitTreeState)
	}
	if info.BuildDate != BuildDate {
		t.Errorf("BuildDate = %q, want %q", info.BuildDate, BuildDate)
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}
	wantPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if info.Platform != wantPlatform {
		t.Errorf("Platform = %q, want %q", info.Platform, wantPlatform)
	}
}

func TestGetInfo_JSON(t *testing.T) {
	info := GetInfo()
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded Info
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if decoded.Version != info.Version {
		t.Errorf("round-trip Version = %q, want %q", decoded.Version, info.Version)
	}
}

func TestHumanVersion(t *testing.T) {
	hv := HumanVersion()

	if !strings.HasPrefix(hv, "custos v") {
		t.Errorf("HumanVersion() = %q, want prefix %q", hv, "custos v")
	}
	if !strings.Contains(hv, Version) && !strings.Contains(hv, strings.TrimPrefix(Version, "v")) {
		t.Errorf("HumanVersion() = %q, does not contain version %q", hv, Version)
	}
}

func TestHumanVersion_TruncatesLongCommit(t *testing.T) {
	orig := GitCommit
	defer func() { GitCommit = orig }()

	GitCommit = "abcdef1234567890"
	hv := HumanVersion()

	if !strings.Contains(hv, "abcdef1") {
		t.Errorf("HumanVersion() = %q, want truncated commit %q", hv, "abcdef1")
	}
	if strings.Contains(hv, "abcdef1234567890") {
		t.Errorf("HumanVersion() = %q, should not contain full commit", hv)
	}
}

func TestHumanVersion_ShortCommit(t *testing.T) {
	orig := GitCommit
	defer func() { GitCommit = orig }()

	GitCommit = "abc"
	hv := HumanVersion()

	if !strings.Contains(hv, "abc") {
		t.Errorf("HumanVersion() = %q, want short commit %q", hv, "abc")
	}
}

func TestHumanVersion_StripsLeadingV(t *testing.T) {
	orig := Version
	defer func() { Version = orig }()

	Version = "v1.2.3"
	hv := HumanVersion()

	if !strings.Contains(hv, "custos v1.2.3") {
		t.Errorf("HumanVersion() = %q, want 'custos v1.2.3' (no double v)", hv)
	}
	if strings.Contains(hv, "custos vv") {
		t.Errorf("HumanVersion() = %q, should not contain 'vv'", hv)
	}
}
