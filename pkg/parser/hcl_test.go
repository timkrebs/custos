package parser

import (
	"path/filepath"
	"testing"
)

func TestParsePolicyFile(t *testing.T) {
	tests := []struct {
		name      string
		file      string // relative to testdata/
		wantPaths int
		wantErr   bool
		check     func(t *testing.T, p *Policy)
	}{
		{
			name:      "basic capabilities",
			file:      "config.hcl",
			wantPaths: 3,
			check: func(t *testing.T, p *Policy) {
				// Exact path
				if p.Paths[0].Path != "secret/foo" {
					t.Errorf("path[0] = %q, want %q", p.Paths[0].Path, "secret/foo")
				}
				if len(p.Paths[0].Capabilities) != 1 || p.Paths[0].Capabilities[0] != "read" {
					t.Errorf("path[0] capabilities = %v, want [read]", p.Paths[0].Capabilities)
				}
				// Glob pattern preserved
				if p.Paths[1].Path != "secret/bar/*" {
					t.Errorf("path[1] = %q, want %q", p.Paths[1].Path, "secret/bar/*")
				}
				// Prefix glob preserved
				if p.Paths[2].Path != "secret/zip-*" {
					t.Errorf("path[2] = %q, want %q", p.Paths[2].Path, "secret/zip-*")
				}
			},
		},
		{
			name:    "empty path errors",
			file:    "", // will trigger read error
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.file != "" {
				path = filepath.Join("..", "..", "testdata", tt.file)
			}
			p, err := ParsePolicyFile(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePolicyFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(p.Paths) != tt.wantPaths {
				t.Fatalf("got %d paths, want %d", len(p.Paths), tt.wantPaths)
			}
			if tt.check != nil {
				tt.check(t, p)
			}
		})
	}
}

func TestParsePolicy_AllFields(t *testing.T) {
	src := []byte(`
path "secret/restricted" {
  capabilities       = ["create"]
  allowed_parameters = {
    "foo" = []
    "bar" = ["zip", "zap"]
  }
  denied_parameters = {
    "baz" = []
  }
  required_parameters = ["foo"]
  min_wrapping_ttl    = "1s"
  max_wrapping_ttl    = "90s"
}
`)
	p, err := ParsePolicy("test.hcl", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Paths) != 1 {
		t.Fatalf("got %d paths, want 1", len(p.Paths))
	}
	r := p.Paths[0]
	if r.MinWrappingTTL != "1s" {
		t.Errorf("MinWrappingTTL = %q, want %q", r.MinWrappingTTL, "1s")
	}
	if r.MaxWrappingTTL != "90s" {
		t.Errorf("MaxWrappingTTL = %q, want %q", r.MaxWrappingTTL, "90s")
	}
	if len(r.AllowedParameters) != 2 {
		t.Errorf("AllowedParameters has %d keys, want 2", len(r.AllowedParameters))
	}
	if vals := r.AllowedParameters["bar"]; len(vals) != 2 || vals[0] != "zip" {
		t.Errorf("AllowedParameters[bar] = %v, want [zip zap]", vals)
	}
	if len(r.DeniedParameters) != 1 {
		t.Errorf("DeniedParameters has %d keys, want 1", len(r.DeniedParameters))
	}
	if len(r.RequiredParameters) != 1 || r.RequiredParameters[0] != "foo" {
		t.Errorf("RequiredParameters = %v, want [foo]", r.RequiredParameters)
	}
}

func TestParsePolicy_MalformedHCL(t *testing.T) {
	_, err := ParsePolicy("bad.hcl", []byte(`path "x" { capabilities = `))
	if err == nil {
		t.Fatal("expected error for malformed HCL")
	}
}

func TestParsePolicy_GlobPatterns(t *testing.T) {
	src := []byte(`
path "secret/+/teamb" {
  capabilities = ["read"]
}
path "secret/+/+/teamb" {
  capabilities = ["read"]
}
path "secret/*" {
  capabilities = ["list"]
}
`)
	p, err := ParsePolicy("globs.hcl", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify glob patterns are preserved as-is (not expanded)
	wantPaths := []string{"secret/+/teamb", "secret/+/+/teamb", "secret/*"}
	for i, want := range wantPaths {
		if p.Paths[i].Path != want {
			t.Errorf("path[%d] = %q, want %q", i, p.Paths[i].Path, want)
		}
	}
}
