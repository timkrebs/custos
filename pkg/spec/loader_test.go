package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string // substring match, empty = no error
		check   func(t *testing.T, s *Spec)
	}{
		{
			name: "valid minimal spec",
			yaml: `
suite: test-suite
tests:
  - name: read secret
    path: secret/foo
    capabilities: [read]
    expect: allow
`,
			check: func(t *testing.T, s *Spec) { /* assert fields */ },
		},
		{name: "missing suite", yaml: "tests:\n  - name: x\n    path: p\n    capabilities: [read]\n    expect: allow", wantErr: "missing required field: suite"},
		{name: "no tests", yaml: "suite: x\ntests: []", wantErr: "at least one test"},
		{name: "invalid capability", yaml: "suite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [bogus]\n    expect: allow", wantErr: "invalid capability"},
		{name: "invalid expect", yaml: "suite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: maybe", wantErr: "expect must be"},
		{name: "missing test name", yaml: "suite: x\ntests:\n  - path: p\n    capabilities: [read]\n    expect: allow", wantErr: "missing required field: name"},
		{name: "unknown top-level field", yaml: "suite: x\nbogus: y\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow", wantErr: "field bogus not found"},
		{name: "typo in test field", yaml: "suite: x\ntests:\n  - name: t\n    path: p\n    capabilties: [read]\n    expect: allow", wantErr: "field capabilties not found"},
		{name: "duplicate test name", yaml: "suite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow\n  - name: t\n    path: q\n    capabilities: [read]\n    expect: allow", wantErr: "duplicate test name"},
		{name: "analyze missing check", yaml: "suite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow\nanalyze:\n  - severity: error", wantErr: "missing required field: check"},
		{name: "analyze invalid severity", yaml: "suite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow\nanalyze:\n  - check: c\n    severity: critical", wantErr: "invalid severity"},
		{name: "analyze bad min_coverage", yaml: "suite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow\nanalyze:\n  - check: c\n    min_coverage: \"abc\"", wantErr: "invalid min_coverage"},
		{name: "analyze min_coverage out of range", yaml: "suite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow\nanalyze:\n  - check: c\n    min_coverage: \"150\"", wantErr: "min_coverage must be in [0, 100]"},
		{
			name: "analyze min_coverage numeric",
			yaml: "suite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow\nanalyze:\n  - check: c\n    min_coverage: 80",
			check: func(t *testing.T, s *Spec) {
				if got := s.Analyze[0].MinCoverage.Float(); got != 80 {
					t.Errorf("MinCoverage = %v, want 80", got)
				}
			},
		},
		{
			name: "analyze min_coverage percent string",
			yaml: "suite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow\nanalyze:\n  - check: c\n    min_coverage: \"80%\"",
			check: func(t *testing.T, s *Spec) {
				if got := s.Analyze[0].MinCoverage.Float(); got != 80 {
					t.Errorf("MinCoverage = %v, want 80", got)
				}
			},
		},
		{
			name: "version v1 accepted",
			yaml: "version: v1\nsuite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow",
			check: func(t *testing.T, s *Spec) {
				if s.Version != "v1" {
					t.Errorf("Version = %q, want v1", s.Version)
				}
			},
		},
		{
			name: "version empty accepted for back-compat",
			yaml: "suite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow",
			check: func(t *testing.T, s *Spec) {
				if s.Version != "" {
					t.Errorf("Version = %q, want empty", s.Version)
				}
			},
		},
		{name: "version unknown rejected", yaml: "version: v99\nsuite: x\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow", wantErr: "unsupported spec version"},
		{
			name: "policies as scalar strings",
			yaml: "suite: x\npolicies:\n  - a.hcl\n  - b.hcl\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow",
			check: func(t *testing.T, s *Spec) {
				if len(s.Policies) != 2 || s.Policies[0].Path != "a.hcl" || s.Policies[1].Path != "b.hcl" {
					t.Errorf("Policies = %+v, want [{a.hcl} {b.hcl}]", s.Policies)
				}
			},
		},
		{
			name: "policies as mappings still work",
			yaml: "suite: x\npolicies:\n  - path: a.hcl\ntests:\n  - name: t\n    path: p\n    capabilities: [read]\n    expect: allow",
			check: func(t *testing.T, s *Spec) {
				if len(s.Policies) != 1 || s.Policies[0].Path != "a.hcl" {
					t.Errorf("Policies = %+v, want [{a.hcl}]", s.Policies)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := Load([]byte(tt.yaml))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got err=%v, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, s)
			}
		})
	}
}

func TestValidate_JoinsMultipleErrors(t *testing.T) {
	y := `suite: ""
tests:
  - name: t
    path: p
    capabilities: [bogus]
    expect: maybe
`
	_, err := Load([]byte(y))
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"missing required field: suite", "invalid capability", "expect must be"} {
		if !strings.Contains(msg, want) {
			t.Errorf("joined error missing %q; got: %s", want, msg)
		}
	}
}

func TestLoadFile_MissingFile(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("expected error for missing spec file")
	}
	if !strings.Contains(err.Error(), "reading spec file") {
		t.Errorf("err = %q, want 'reading spec file'", err.Error())
	}
}

func TestLoad_EmptyBytes(t *testing.T) {
	_, err := Load(nil)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestPolicyRef_SequenceFormRejected(t *testing.T) {
	y := `suite: x
policies:
  - [a, b]
tests:
  - name: t
    path: p
    capabilities: [read]
    expect: allow
`
	_, err := Load([]byte(y))
	if err == nil {
		t.Fatal("expected error for sequence-form policies entry")
	}
	if !strings.Contains(err.Error(), "expected string or mapping") {
		t.Errorf("err = %q, want mention of string or mapping", err.Error())
	}
}

func TestPercentage_MappingRejected(t *testing.T) {
	y := `suite: x
tests:
  - name: t
    path: p
    capabilities: [read]
    expect: allow
analyze:
  - check: c
    min_coverage:
      nested: value
`
	_, err := Load([]byte(y))
	if err == nil {
		t.Fatal("expected error for mapping-form min_coverage")
	}
	if !strings.Contains(err.Error(), "invalid min_coverage") {
		t.Errorf("err = %q, want 'invalid min_coverage'", err.Error())
	}
}

func TestLoadFile_ResolvesRelativePolicyPaths(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(`suite: x
policies:
  - policy.hcl
tests:
  - name: t
    path: p
    capabilities: [read]
    expect: allow
`), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadFile(specPath)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	want := filepath.Join(dir, "policy.hcl")
	if s.Policies[0].Path != want {
		t.Errorf("Policies[0].Path = %q, want %q", s.Policies[0].Path, want)
	}
}

func TestLoadFile_ResolvesSiblingDirectoryLayout(t *testing.T) {
	root := t.TempDir()
	specDir := filepath.Join(root, "specs")
	if err := os.Mkdir(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(specDir, "svc.yaml")
	if err := os.WriteFile(specPath, []byte(`suite: x
policies:
  - ../policies/svc.hcl
tests:
  - name: t
    path: p
    capabilities: [read]
    expect: allow
`), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadFile(specPath)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	want := filepath.Join(specDir, "..", "policies", "svc.hcl")
	if s.Policies[0].Path != want {
		t.Errorf("Policies[0].Path = %q, want %q", s.Policies[0].Path, want)
	}
}

func TestLoadFile_RejectsEmptyPolicyPath(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(`suite: x
policies:
  - ""
tests:
  - name: t
    path: p
    capabilities: [read]
    expect: allow
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFile(specPath)
	if err == nil || !strings.Contains(err.Error(), "empty path") {
		t.Errorf("err = %v, want 'empty path'", err)
	}
}

func TestLoadFile_AbsolutePolicyPathAllowed(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	abs := filepath.Join(dir, "..", "somewhere", "policy.hcl")
	if err := os.WriteFile(specPath, []byte(`suite: x
policies:
  - `+abs+`
tests:
  - name: t
    path: p
    capabilities: [read]
    expect: allow
`), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadFile(specPath)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if s.Policies[0].Path != filepath.Clean(abs) {
		t.Errorf("Policies[0].Path = %q, want %q", s.Policies[0].Path, filepath.Clean(abs))
	}
}
