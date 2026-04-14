package spec

import (
	"bytes"
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

	// Resolve relative policy paths against the spec file's directory.
	// Sibling-directory layouts (e.g. ../policies/foo.hcl from a specs/ dir)
	// are a legitimate convention, so `..` segments are allowed — the spec
	// author is the user, not an untrusted input.
	dir := filepath.Dir(path)
	for i, p := range spec.Policies {
		if p.Path == "" {
			return nil, fmt.Errorf("%s: policies[%d]: empty path", path, i)
		}
		if filepath.IsAbs(p.Path) {
			spec.Policies[i].Path = filepath.Clean(p.Path)
			continue
		}
		spec.Policies[i].Path = filepath.Join(dir, filepath.Clean(p.Path))
	}

	return spec, nil
}

// Load parses and validates a spec from raw YAML bytes.
func Load(data []byte) (*Spec, error) {
	var s Spec
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if err := validate(&s); err != nil {
		return nil, fmt.Errorf("validating spec: %w", err)
	}
	return &s, nil
}
