# Overprivileged policy — intentionally dangerous for scanner testing.
# This policy contains multiple security anti-patterns that custos
# scan should detect.

# Anti-pattern: wildcard path with all capabilities
path "secret/*" {
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# Anti-pattern: sudo on non-sys path
path "database/config/*" {
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# Anti-pattern: root token creation
path "auth/token/create" {
  capabilities = ["create", "update"]
}

# Anti-pattern: policy escalation — can modify own policies
path "sys/policies/acl/*" {
  capabilities = ["create", "read", "update", "delete"]
}

# Anti-pattern: can unseal vault
path "sys/seal" {
  capabilities = ["sudo"]
}

path "sys/unseal" {
  capabilities = ["sudo", "update"]
}

# Anti-pattern: can destroy secret versions permanently
path "secret/destroy/*" {
  capabilities = ["update"]
}

path "secret/metadata/*" {
  capabilities = ["delete"]
}
