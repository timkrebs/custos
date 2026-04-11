package parser

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

type Policy struct {
	Filepath string `hcl:"filepath"`
	Paths    []PathRule
}

type ProcessConfig struct {
	Type    string   `hcl:"type,label"`
	Command []string `hcl:"command"`
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

func ParsePolicy(filename string, src []byte) (*Policy, error) {
	parse := hclparse.NewParser()
	file, diags := parse.ParseHCL(src, filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL: %s", diags.Error())
	}

	var raw hclPolicy
	diags = gohcl.DecodeBody(file.Body, nil, &raw)
	if diags.HasErrors() {
		return nil, fmt.Errorf("decoding policy: %s", diags.Error())
	}

	policy := &Policy{Filepath: filename}
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

		// Process remaining fields for allowed/denied parameters
		if rp.Remain != nil {
			attrs, diags := rp.Remain.JustAttributes()
			if diags.HasErrors() {
				log.Printf("Error processing remaining fields for path %s: %s", rp.Path, diags.Error())
				continue
			}

			for name, attr := range attrs {
				val, diags := attr.Expr.Value(nil)
				if diags.HasErrors() {
					log.Printf("Error evaluating attribute %s for path %s: %s", name, rp.Path, diags.Error())
					return nil, fmt.Errorf("evaluating %q in path %q: %s", name, rp.Path, diags.Error())
				}

				switch name {
				case "allowed_parameters":
					pr.AllowedParameters = decodeParamMap(val)
					log.Printf("Decoded allowed_parameters for path %s: %v", rp.Path, pr.AllowedParameters)
				case "denied_parameters":
					pr.DeniedParameters = decodeParamMap(val)
					log.Printf("Decoded denied_parameters for path %s: %v", rp.Path, pr.DeniedParameters)
				case "required_parameters":
					pr.RequiredParameters = decodeStringList(val)
					log.Printf("Decoded required_parameters for path %s: %v", rp.Path, pr.RequiredParameters)
				default:
					log.Printf("Unknown attribute %s in path %s", name, rp.Path)
				}
			}
		}

		policy.Paths = append(policy.Paths, pr)
	}
	return policy, nil
}

func decodeParamMap(val cty.Value) map[string][]string {
	if val.IsNull() || !val.IsKnown() {
		return nil
	}
	result := make(map[string][]string)
	for key, v := range val.AsValueMap() {
		var values []string
		if v.Type().IsTupleType() || v.Type().IsListType() || v.Type().IsSetType() {
			for it := v.ElementIterator(); it.Next(); {
				_, elem := it.Element()
				values = append(values, elem.AsString())
			}
		}
		result[key] = values
	}
	return result
}

func decodeStringList(val cty.Value) []string {
	if val.IsNull() || !val.IsKnown() {
		return nil
	}
	var result []string
	for it := val.ElementIterator(); it.Next(); {
		_, elem := it.Element()
		result = append(result, elem.AsString())
	}
	return result
}
