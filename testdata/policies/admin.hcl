# Admin policy — broad access for platform operators.
# Grants wide read/list on secrets, full sys/ management,
# and auth method administration.

# Full access to all secrets
path "secret/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# Vault system backend — seal, unseal, health, leader
path "sys/*" {
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# Auth method management
path "auth/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# Policy management (dangerous — allows self-escalation)
path "sys/policies/acl/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# Identity management
path "identity/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# Audit device management
path "sys/audit/*" {
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
