# Read-only policy — minimal access for monitoring and auditing.
# Can read and list secrets but cannot create, update, or delete.

# Read and list all secrets
path "secret/*" {
  capabilities = ["read", "list"]
}

# Read system health (no sudo)
path "sys/health" {
  capabilities = ["read"]
}

# Read own token info
path "auth/token/lookup-self" {
  capabilities = ["read"]
}

# Read policy list (but not modify)
path "sys/policies/acl" {
  capabilities = ["read", "list"]
}

# Read audit log configuration
path "sys/audit" {
  capabilities = ["read"]
}

# Cubbyhole — every token gets its own
path "cubbyhole/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
