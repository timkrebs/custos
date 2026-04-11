package spec

import (
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
