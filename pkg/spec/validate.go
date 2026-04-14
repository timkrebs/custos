package spec

import (
	"errors"
	"fmt"

	"github.com/timkrebs/custos/pkg/vaultpolicy"
)

var validExpect = map[string]bool{
	"allow": true, "deny": true,
}

var validSeverity = map[string]bool{
	"error": true, "warning": true, "warn": true, "info": true,
}

func validate(s *Spec) error {
	var errs []error

	if s.Version != "" && s.Version != CurrentVersion {
		errs = append(errs, fmt.Errorf("unsupported spec version %q (want %q or empty)", s.Version, CurrentVersion))
	}
	if s.Suite == "" {
		errs = append(errs, errors.New("missing required field: suite"))
	}
	if len(s.Tests) == 0 {
		errs = append(errs, errors.New("spec must contain at least one test"))
	}

	seen := make(map[string]int, len(s.Tests))
	for i, tc := range s.Tests {
		prefix := fmt.Sprintf("test[%d]", i)
		if tc.Name == "" {
			errs = append(errs, fmt.Errorf("%s: missing required field: name", prefix))
		} else {
			prefix = fmt.Sprintf("test[%d] %q", i, tc.Name)
			if j, dup := seen[tc.Name]; dup {
				errs = append(errs, fmt.Errorf("%s: duplicate test name (also at test[%d])", prefix, j))
			} else {
				seen[tc.Name] = i
			}
		}
		if tc.Path == "" {
			errs = append(errs, fmt.Errorf("%s: missing required field: path", prefix))
		}
		if len(tc.Capabilities) == 0 {
			errs = append(errs, fmt.Errorf("%s: missing required field: capabilities", prefix))
		}
		for _, cap := range tc.Capabilities {
			if !vaultpolicy.IsValidCapability(cap) {
				errs = append(errs, fmt.Errorf("%s: invalid capability %q", prefix, cap))
			}
		}
		if !validExpect[tc.Expect] {
			errs = append(errs, fmt.Errorf("%s: expect must be \"allow\" or \"deny\", got %q", prefix, tc.Expect))
		}
	}

	for i, a := range s.Analyze {
		prefix := fmt.Sprintf("analyze[%d]", i)
		if a.Check == "" {
			errs = append(errs, fmt.Errorf("%s: missing required field: check", prefix))
		}
		if a.Severity != "" && !validSeverity[a.Severity] {
			errs = append(errs, fmt.Errorf("%s: invalid severity %q (want error|warning|info)", prefix, a.Severity))
		}
		if a.MinCoverage != nil {
			v := a.MinCoverage.Float()
			if v < 0 || v > 100 {
				errs = append(errs, fmt.Errorf("%s: min_coverage must be in [0, 100], got %v", prefix, v))
			}
		}
	}

	return errors.Join(errs...)
}
