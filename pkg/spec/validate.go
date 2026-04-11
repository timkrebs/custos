package spec

import (
	"fmt"
)

// Valid Vault capabilities
var validCapabilities = map[string]bool{
	"create": true, "read": true, "update": true,
	"patch": true, "delete": true, "list": true,
	"sudo": true, "deny": true, "subscribe": true, "recover": true,
}

var validExpect = map[string]bool{
	"allow": true, "deny": true,
}

func validate(s *Spec) error {
	if s.Suite == "" {
		return fmt.Errorf("missing required field: suite")
	}
	if len(s.Tests) == 0 {
		return fmt.Errorf("spec must contain at least one test")
	}
	for i, tc := range s.Tests {
		if tc.Name == "" {
			return fmt.Errorf("test[%d]: missing required field: name", i)
		}
		if tc.Path == "" {
			return fmt.Errorf("test[%d] %q: missing required field: path", i, tc.Name)
		}
		if len(tc.Capabilities) == 0 {
			return fmt.Errorf("test[%d] %q: missing required field: capabilities", i, tc.Name)
		}
		for _, cap := range tc.Capabilities {
			if !validCapabilities[cap] {
				return fmt.Errorf("test[%d] %q: invalid capability %q", i, tc.Name, cap)
			}
		}
		if !validExpect[tc.Expect] {
			return fmt.Errorf("test[%d] %q: expect must be \"allow\" or \"deny\", got %q", i, tc.Name, tc.Expect)
		}
	}
	return nil
}
