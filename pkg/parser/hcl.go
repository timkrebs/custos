package parser

import (
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

type Policy struct {
	Filepath string
	Paths    []PathRule
}

type PathRule struct {
	Path               string
	Capabilities       []string
	AllowedParameters  map[string][]string
	DeniedParameters   map[string][]string
	RequiredParameters []string
	MinWrappingTTL     string
	MaxWrappingTTL     string
}

// hclPolicy is the top-level HCL decode target.
type hclPolicy struct {
	Path []hclPathRule `hcl:"path,block"`
}

type hclPathRule struct {
	Path           string   `hcl:"path,label"`
	Capabilities   []string `hcl:"capabilities"`
	MinWrappingTTL *string  `hcl:"min_wrapping_ttl,optional"`
	MaxWrappingTTL *string  `hcl:"max_wrapping_ttl,optional"`
	Remain         hcl.Body `hcl:",remain"` // Capture remaining fields for further processing
}

func ParsePolicyFile(path string) (*Policy, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading policy file: %w", err)
	}
	return ParsePolicy(path, src)
}

// ParsePolicyFileDiag is the diagnostics-returning variant of ParsePolicyFile.
// The returned *hclparse.Parser's Files() map can be passed to
// hcl.NewDiagnosticTextWriter for pretty source-annotated error output.
func ParsePolicyFileDiag(path string) (*Policy, *hclparse.Parser, hcl.Diagnostics) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, hclparse.NewParser(), hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "cannot read policy file",
			Detail:   err.Error(),
		}}
	}
	return ParsePolicyDiag(path, src)
}

// ParsePolicy parses an HCL policy and returns a flat error. Callers that
// want rich source-annotated diagnostics should use ParsePolicyDiag instead.
func ParsePolicy(filename string, src []byte) (*Policy, error) {
	p, _, diags := ParsePolicyDiag(filename, src)
	if diags.HasErrors() {
		return nil, errors.New(diags.Error())
	}
	return p, nil
}

// ParsePolicyDiag parses an HCL policy and returns rich diagnostics plus the
// underlying parser. The parser's Files() map can be used with
// hcl.NewDiagnosticTextWriter to render errors with file:line:col context.
func ParsePolicyDiag(filename string, src []byte) (*Policy, *hclparse.Parser, hcl.Diagnostics) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(src, filename)
	if diags.HasErrors() {
		return nil, parser, diags
	}

	var raw hclPolicy
	if d := gohcl.DecodeBody(file.Body, nil, &raw); d.HasErrors() {
		return nil, parser, d
	}

	policy := &Policy{Filepath: filename}
	var allDiags hcl.Diagnostics
	for _, rp := range raw.Path {
		pr := PathRule{
			Path:         rp.Path,
			Capabilities: rp.Capabilities,
		}
		if rp.MinWrappingTTL != nil {
			pr.MinWrappingTTL = *rp.MinWrappingTTL
		}
		if rp.MaxWrappingTTL != nil {
			pr.MaxWrappingTTL = *rp.MaxWrappingTTL
		}

		if rp.Remain != nil {
			attrs, d := rp.Remain.JustAttributes()
			allDiags = append(allDiags, d...)

			for name, attr := range attrs {
				rng := attr.Expr.Range()
				val, vd := attr.Expr.Value(nil)
				if vd.HasErrors() {
					allDiags = append(allDiags, vd...)
					continue
				}

				switch name {
				case "allowed_parameters":
					m, md := decodeParamMap(name, val, rng)
					allDiags = append(allDiags, md...)
					pr.AllowedParameters = m
				case "denied_parameters":
					m, md := decodeParamMap(name, val, rng)
					allDiags = append(allDiags, md...)
					pr.DeniedParameters = m
				case "required_parameters":
					l, ld := decodeStringList(name, val, rng)
					allDiags = append(allDiags, ld...)
					pr.RequiredParameters = l
				}
			}
		}

		policy.Paths = append(policy.Paths, pr)
	}

	return policy, parser, allDiags
}

func decodeParamMap(name string, val cty.Value, rng hcl.Range) (map[string][]string, hcl.Diagnostics) {
	if val.IsNull() || !val.IsKnown() {
		return nil, nil
	}
	t := val.Type()
	if !t.IsObjectType() && !t.IsMapType() {
		return nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "invalid parameter map",
			Detail:   fmt.Sprintf("%q must be a map of string to list of strings, got %s", name, t.FriendlyName()),
			Subject:  &rng,
		}}
	}
	var diags hcl.Diagnostics
	result := make(map[string][]string)
	for key, v := range val.AsValueMap() {
		vt := v.Type()
		if !(vt.IsTupleType() || vt.IsListType() || vt.IsSetType()) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "invalid parameter value",
				Detail:   fmt.Sprintf("%q[%q] must be a list, got %s", name, key, vt.FriendlyName()),
				Subject:  &rng,
			})
			continue
		}
		var values []string
		ok := true
		for it := v.ElementIterator(); it.Next(); {
			_, elem := it.Element()
			if elem.Type() != cty.String {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "invalid parameter element",
					Detail:   fmt.Sprintf("%q[%q] element must be a string, got %s", name, key, elem.Type().FriendlyName()),
					Subject:  &rng,
				})
				ok = false
				break
			}
			values = append(values, elem.AsString())
		}
		if ok {
			result[key] = values
		}
	}
	return result, diags
}

func decodeStringList(name string, val cty.Value, rng hcl.Range) ([]string, hcl.Diagnostics) {
	if val.IsNull() || !val.IsKnown() {
		return nil, nil
	}
	t := val.Type()
	if !(t.IsTupleType() || t.IsListType() || t.IsSetType()) {
		return nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "invalid list",
			Detail:   fmt.Sprintf("%q must be a list of strings, got %s", name, t.FriendlyName()),
			Subject:  &rng,
		}}
	}
	var diags hcl.Diagnostics
	var result []string
	for it := val.ElementIterator(); it.Next(); {
		_, elem := it.Element()
		if elem.Type() != cty.String {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "invalid list element",
				Detail:   fmt.Sprintf("%q element must be a string, got %s", name, elem.Type().FriendlyName()),
				Subject:  &rng,
			})
			continue
		}
		result = append(result, elem.AsString())
	}
	return result, diags
}
