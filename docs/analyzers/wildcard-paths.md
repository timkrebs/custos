---
description: Flags overly broad wildcard paths that grant mutation and read together.
icon: asterisk
---

# wildcard\_paths

**Severity:** warning (configurable)

## What it flags

Any policy rule whose path ends in `*` and whose capabilities list contains three or more entries.

```hcl
path "secret/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
```

This single rule grants the entity full mutation access to every path under `secret/`. The analyzer emits:

```
wildcard path "secret/*" grants 5 capabilities
  (create, read, update, delete, list); narrow the path or split by access level
  at policies/overprivileged.hcl:6
```

## Why it matters

Broad wildcard paths are the single most common cause of accidental overprivilege in Vault deployments. They happen because:

- The policy was written before a secrets backend's path structure was settled and never tightened.
- A "temporary" broad grant was never revisited.
- A team copy-pasted from a Vault tutorial that used `secret/*` for brevity.

The pattern is dangerous because it combines read *and* mutation access at the broadest possible scope. Read-only wildcards (`[read, list]`) are relatively safe; read-plus-write wildcards are how you accidentally grant service A the ability to delete service B's secrets.

## Why three capabilities

Two capabilities are usually safe: `[read, list]` is the canonical read-only grant and triggers no warning. Three capabilities almost always implies at least one mutation capability alongside read, which is the risk.

## How to fix it

The remediation is to narrow the path to the service or role that actually needs the access:

```hcl
# Before: broad wildcard
path "secret/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# After: scoped to a specific service
path "secret/data/payment-svc/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
```

If multiple services share a parent namespace, write one rule per service rather than one wildcard covering them all. The extra verbosity is worth it.

## When the finding is acceptable

A broad wildcard can be legitimate for a vault administrator policy or a break-glass role. For those cases, whitelist the specific rule path in the spec:

```yaml
analyze:
  - check: wildcard_paths
    allow_paths:
      - secret/*     # admin-legacy policy, reviewed quarterly
```

Prefer narrow whitelisting over disabling the check entirely.

## Configuration

```yaml
analyze:
  - check: wildcard_paths
    disabled: false
    allow_paths:
      - sys/*
      - secret/admin/*
    severity: error    # promote to blocking
```

## Implementation

The check walks every parsed `PathRule`, looks for a trailing `*`, and counts capabilities excluding `deny`. Read-only paths (`[read]`, `[list]`, `[read, list]`) are never flagged even if the wildcard is broad, because the composed blast radius is limited to data exposure, not mutation. Adjust your threat model accordingly.
