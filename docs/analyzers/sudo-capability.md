---
description: Flags sudo grants on paths outside the system and token backends.
icon: user-shield
---

# sudo\_capability

**Severity:** error (configurable)

## What it flags

Any rule that includes the `sudo` capability on a path that is not under `sys/` or `auth/token/`.

```hcl
path "database/config/*" {
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
```

The analyzer emits:

```
sudo capability on "database/config/*" is outside sys/ and auth/token/;
sudo should only be used with privileged system endpoints
  at policies/overprivileged.hcl:12
```

## Why it matters

The `sudo` capability exists to grant access to a small set of Vault endpoints that require elevated privileges — things like `sys/unseal`, `sys/rotate`, and `sys/audit`. It does nothing on secret engines like `database/*`, `kv/*`, or `transit/*`. When `sudo` appears on one of those paths, it is almost always because somebody:

- Copy-pasted a rule from a Vault tutorial without reading what `sudo` means.
- Thought `sudo` was a generic "grant everything" capability. It is not.
- Was cargo-culting from another policy.

The finding is an error, not a warning, because the wrong understanding of what `sudo` does often comes with other misconfigurations.

## What `sudo` actually grants

`sudo` unlocks access to endpoints marked "root-protected" in Vault. These are mostly under `sys/` (system management) and `auth/token/` (token lifecycle operations like renewal on behalf of others). For regular secret engines, `sudo` is silently ignored — but the fact that someone wrote it suggests they expected it to do something, and that expectation is worth investigating.

## How to fix it

Remove the `sudo` capability from the rule. If the intent was to grant full CRUD access, the other capabilities in the list already do that:

```hcl
# Before
path "database/config/*" {
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

# After
path "database/config/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
```

If you genuinely need `sudo` on a specific non-system path — for example, a secret engine that exposes a sudo-protected rotation endpoint — whitelist just that path.

## Legitimate exceptions

A small number of secrets engines expose sudo-protected endpoints. `database/config/rotate` is the canonical example: it rotates the root credentials used by Vault to manage database dynamic secrets, and it requires `sudo`. For those cases:

```yaml
analyze:
  - check: sudo_capability
    allow_paths:
      - database/config/rotate
      - pki/root/rotate/internal
```

Narrow whitelisting beats blanket disabling.

## Configuration

```yaml
analyze:
  - check: sudo_capability
    disabled: false
    allow_paths:
      - database/config/rotate
    severity: warning   # downgrade if the team prefers
```

## Implementation

The check walks every rule and tests whether `sudo` is in the capabilities list. If so, it checks whether the rule path starts with `sys/` or `auth/token/` — if not, it is flagged. The prefix check is string-based and accepts both exact matches and prefixed paths (`sys/seal`, `sys/policies/acl/admin`, `auth/token/renew-self`).
