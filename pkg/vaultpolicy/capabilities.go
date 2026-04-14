// Package vaultpolicy holds the semantic vocabulary shared between the HCL
// parser and the spec validator — primarily the set of valid Vault policy
// capabilities. Keeping this in one place prevents the two layers from
// drifting out of sync when new Vault capabilities are added.
package vaultpolicy

// Capabilities is the set of valid Vault policy capability names.
// See https://developer.hashicorp.com/vault/docs/concepts/policies#capabilities
var Capabilities = map[string]bool{
	"create":    true,
	"read":      true,
	"update":    true,
	"patch":     true,
	"delete":    true,
	"list":      true,
	"sudo":      true,
	"deny":      true,
	"subscribe": true,
	"recover":   true,
}

// IsValidCapability reports whether name is a recognized Vault capability.
func IsValidCapability(name string) bool {
	return Capabilities[name]
}
