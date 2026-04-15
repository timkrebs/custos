package spec

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// CurrentVersion is the only spec schema version currently recognized.
// An empty version is accepted for back-compat with pre-versioned specs.
const CurrentVersion = "v1"

// Spec is the top-level test specification
type Spec struct {
	Version  string         `yaml:"version,omitempty"`
	Suite    string         `yaml:"suite"`
	Policies []PolicyRef    `yaml:"policies"`
	Tests    []TestCase     `yaml:"tests"`
	Analyze  []AnalyzeCheck `yaml:"analyze,omitempty"`
}

type PolicyRef struct {
	Path string `yaml:"path"`
}

// UnmarshalYAML accepts either a scalar form (`- foo.hcl`) or a mapping
// form (`- path: foo.hcl`).
func (p *PolicyRef) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		p.Path = node.Value
		return nil
	case yaml.MappingNode:
		type raw PolicyRef
		var r raw
		if err := node.Decode(&r); err != nil {
			return err
		}
		*p = PolicyRef(r)
		return nil
	default:
		return fmt.Errorf("policies entry: expected string or mapping at line %d", node.Line)
	}
}

type TestCase struct {
	Name         string   `yaml:"name"`
	Path         string   `yaml:"path"`
	Capabilities []string `yaml:"capabilities"`
	Expect       string   `yaml:"expect"`
}

type AnalyzeCheck struct {
	Check       string      `yaml:"check"`
	WarnOn      string      `yaml:"warn_on,omitempty"`
	MinCoverage *Percentage `yaml:"min_coverage,omitempty"`
	Severity    string      `yaml:"severity,omitempty"`

	// Disabled turns the check off entirely. Useful when a team knows a
	// given rule does not apply to their policy layout and wants to keep
	// noise out of the report without removing the entry from the spec.
	Disabled bool `yaml:"disabled,omitempty"`

	// AllowPaths is the per-check exception list. Entries are matched
	// against the offending policy path with Vault-style glob semantics
	// (trailing `*` = prefix match, `+` = single segment, otherwise
	// exact). An allow_paths entry that matches causes the check to
	// suppress the finding for that path — this is how an operator
	// legitimately whitelists a sudo grant on `sys/unseal` or a wildcard
	// on an internal secret tree.
	AllowPaths []string `yaml:"allow_paths,omitempty"`
}

// Percentage is a float in [0, 100] that accepts both numeric forms (e.g.
// `min_coverage: 80`, `min_coverage: 80.5`) and string forms with an
// optional `%` suffix (e.g. `min_coverage: "80%"`). Parsing failures
// surface at YAML-decode time so downstream code can rely on the value.
type Percentage float64

func (p *Percentage) UnmarshalYAML(node *yaml.Node) error {
	raw := strings.TrimSpace(node.Value)
	if raw == "" {
		return fmt.Errorf("invalid min_coverage: empty value")
	}
	trimmed := strings.TrimSuffix(raw, "%")
	v, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return fmt.Errorf("invalid min_coverage %q: not a number", raw)
	}
	*p = Percentage(v)
	return nil
}

// Float returns the underlying float value (0 if unset).
func (p *Percentage) Float() float64 {
	if p == nil {
		return 0
	}
	return float64(*p)
}
