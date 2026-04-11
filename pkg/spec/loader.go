package spec

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadFile reads and validates a spec YAML file.
// Policy paths are resolved relative to the spec file's directory.
// LoadFile reads and validates a spec YAML file.
func LoadFile(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading spec file: %w", err)
	}

	spec, err := Load(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	// Resolve policy paths relative to spec file
	dir := filepath.Dir(path)
	for i, p := range spec.Policies {
		if !filepath.IsAbs(p.Path) {
			spec.Policies[i].Path = filepath.Join(dir, p.Path)
		}
	}

	return spec, nil
}

// Load parses and validates a spec from raw YAML bytes.
func Load(data []byte) (*Spec, error) {
	var s Spec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if err := validate(&s); err != nil {
		return nil, fmt.Errorf("validating spec: %w", err)
	}
	return &s, nil
}
