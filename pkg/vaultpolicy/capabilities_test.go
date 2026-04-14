package vaultpolicy

import "testing"

func TestIsValidCapability(t *testing.T) {
	for _, c := range []string{"create", "read", "update", "patch", "delete", "list", "sudo", "deny", "subscribe", "recover"} {
		if !IsValidCapability(c) {
			t.Errorf("%q should be valid", c)
		}
	}
	for _, c := range []string{"", "bogus", "READ", "write"} {
		if IsValidCapability(c) {
			t.Errorf("%q should be invalid", c)
		}
	}
}
