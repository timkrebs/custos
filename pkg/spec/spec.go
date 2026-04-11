package spec

// Spec is the top-level test specification
type Spec struct {
	Suite    string         `yaml:"suite"`
	Policies []PolicyRef    `yaml:"policies"`
	Tests    []TestCase     `yaml:"tests"`
	Analyze  []AnalyzeCheck `yaml:"analyze,omitempty"`
}

type PolicyRef struct {
	Path string `yaml:"path"`
}

type TestCase struct {
	Name         string   `yaml:"name"`
	Path         string   `yaml:"path"`
	Capabilities []string `yaml:"capabilities"`
	Expect       string   `yaml:"expect"`
}

type AnalyzeCheck struct {
	Check       string `yaml:"check"`
	WarnOn      string `yaml:"warn_on,omitempty"`
	MinCoverage string `yaml:"min_coverage,omitempty"`
	Severity    string `yaml:"severity,omitempty"`
}
