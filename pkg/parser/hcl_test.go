package parser

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
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

func TestParsePolicy_InvalidStructure(t *testing.T) {
	// Valid HCL but not a valid policy — missing block label triggers DecodeBody error
	src := []byte(`not_a_path_block = "hello"`)
	_, err := ParsePolicy("invalid.hcl", src)
	if err == nil {
		t.Fatal("expected error for invalid policy structure")
	}
}

func TestParsePolicy_UnknownAttribute(t *testing.T) {
	src := []byte(`
path "secret/foo" {
  capabilities     = ["read"]
  unknown_field    = "surprise"
}
`)
	p, err := ParsePolicy("unknown.hcl", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Paths) != 1 {
		t.Fatalf("got %d paths, want 1", len(p.Paths))
	}
	if p.Paths[0].Path != "secret/foo" {
		t.Errorf("path = %q, want %q", p.Paths[0].Path, "secret/foo")
	}
}

func TestDecodeParamMap_Null(t *testing.T) {
	result, diags := decodeParamMap("allowed_parameters", cty.NullVal(cty.DynamicPseudoType), hcl.Range{})
	if result != nil {
		t.Errorf("expected nil for null value, got %v", result)
	}
	if diags.HasErrors() {
		t.Errorf("unexpected diags: %s", diags.Error())
	}
}

func TestDecodeStringList_Null(t *testing.T) {
	result, diags := decodeStringList("required_parameters", cty.NullVal(cty.DynamicPseudoType), hcl.Range{})
	if result != nil {
		t.Errorf("expected nil for null value, got %v", result)
	}
	if diags.HasErrors() {
		t.Errorf("unexpected diags: %s", diags.Error())
	}
}

func TestParsePolicy_NonStringParamElement(t *testing.T) {
	src := []byte(`
path "secret/foo" {
  capabilities       = ["read"]
  allowed_parameters = { "foo" = [1, 2] }
}
`)
	_, err := ParsePolicy("bad.hcl", src)
	if err == nil {
		t.Fatal("expected error for numeric element in allowed_parameters")
	}
	if !strings.Contains(err.Error(), "element must be a string") {
		t.Errorf("error = %q, want to mention string element", err.Error())
	}
}

func TestParsePolicy_AllowedParamsNotMap(t *testing.T) {
	src := []byte(`
path "secret/foo" {
  capabilities       = ["read"]
  allowed_parameters = ["foo", "bar"]
}
`)
	_, err := ParsePolicy("bad.hcl", src)
	if err == nil {
		t.Fatal("expected error when allowed_parameters is a list")
	}
	if !strings.Contains(err.Error(), "must be a map") {
		t.Errorf("error = %q, want to mention map", err.Error())
	}
}

func TestParsePolicy_ParamValueNotList(t *testing.T) {
	src := []byte(`
path "secret/foo" {
  capabilities       = ["read"]
  allowed_parameters = { "foo" = "bar" }
}
`)
	_, err := ParsePolicy("bad.hcl", src)
	if err == nil {
		t.Fatal("expected error when parameter value is a string")
	}
	if !strings.Contains(err.Error(), "must be a list") {
		t.Errorf("error = %q, want to mention list", err.Error())
	}
}

func TestParsePolicy_RequiredParamsNotList(t *testing.T) {
	src := []byte(`
path "secret/foo" {
  capabilities        = ["read"]
  required_parameters = "foo"
}
`)
	_, err := ParsePolicy("bad.hcl", src)
	if err == nil {
		t.Fatal("expected error when required_parameters is not a list")
	}
	if !strings.Contains(err.Error(), "must be a list") {
		t.Errorf("error = %q, want to mention list", err.Error())
	}
}

func TestParsePolicy_UnknownTopLevelBlock(t *testing.T) {
	src := []byte(`foo "x" { capabilities = ["read"] }`)
	_, err := ParsePolicy("bad.hcl", src)
	if err == nil {
		t.Fatal("expected error for unknown top-level block type")
	}
}

func TestParsePolicyFileDiag_MissingFile(t *testing.T) {
	_, parser, diags := ParsePolicyFileDiag("/definitely/not/a/real/path.hcl")
	if !diags.HasErrors() {
		t.Fatal("expected diagnostics for missing file")
	}
	if parser == nil {
		t.Error("expected non-nil parser even on read failure")
	}
	if !strings.Contains(diags.Error(), "cannot read policy file") {
		t.Errorf("diags = %q, want 'cannot read policy file'", diags.Error())
	}
}

func TestParsePolicy_EmptyFile(t *testing.T) {
	p, err := ParsePolicy("empty.hcl", []byte("// just a comment\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Paths) != 0 {
		t.Errorf("got %d paths, want 0", len(p.Paths))
	}
}

func TestParsePolicyDiag_MultipleAttrErrorsAccumulate(t *testing.T) {
	src := []byte(`path "secret/foo" {
  capabilities        = ["read"]
  allowed_parameters  = ["wrong"]
  required_parameters = "also-wrong"
}
`)
	_, _, diags := ParsePolicyDiag("bad.hcl", src)
	if !diags.HasErrors() {
		t.Fatal("expected diagnostics")
	}
	// Aggregation must surface BOTH errors, not just the first.
	// diags.Error() only summarizes ("X; and N other diagnostic(s)"),
	// so iterate the slice directly.
	var sawMap, sawList bool
	for _, d := range diags {
		if strings.Contains(d.Detail, "must be a map") {
			sawMap = true
		}
		if strings.Contains(d.Detail, "must be a list") {
			sawList = true
		}
	}
	if !sawMap {
		t.Errorf("missing map error; got %d diags: %s", len(diags), diags.Error())
	}
	if !sawList {
		t.Errorf("missing list error; got %d diags: %s", len(diags), diags.Error())
	}
}

func TestParsePolicyDiag_SuccessExposesParser(t *testing.T) {
	src := []byte(`path "secret/foo" { capabilities = ["read"] }`)
	p, parser, diags := ParsePolicyDiag("ok.hcl", src)
	if diags.HasErrors() {
		t.Fatalf("unexpected diags: %s", diags.Error())
	}
	if p == nil || len(p.Paths) != 1 {
		t.Fatalf("got %v, want one path", p)
	}
	if _, ok := parser.Files()["ok.hcl"]; !ok {
		t.Errorf("parser.Files() missing ok.hcl, got keys: %v", parser.Files())
	}
}

func TestParsePolicyDiag_ErrorCarriesSourceRange(t *testing.T) {
	src := []byte(`path "secret/foo" {
  capabilities       = ["read"]
  allowed_parameters = { "foo" = [1] }
}
`)
	_, _, diags := ParsePolicyDiag("bad.hcl", src)
	if !diags.HasErrors() {
		t.Fatal("expected diagnostics")
	}
	var found bool
	for _, d := range diags {
		if d.Subject != nil && d.Subject.Filename == "bad.hcl" && d.Subject.Start.Line > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no diagnostic carried a source range for bad.hcl; got %s", diags.Error())
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
